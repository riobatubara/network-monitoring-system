package collector

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// PingTarget executes a native ICMP Echo Request against a target IP or hostname.
// It returns the round-trip latency and any execution errors encountered.
func PingTarget(ctx context.Context, target string, timeout time.Duration) (time.Duration, error) {
	// 1. Resolve the network target to a valid IP address
	ip, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return 0, fmt.Errorf("hostname resolution failed: %w", err)
	}

	// 2. Open an unprivileged UDP ICMP socket listener endpoint
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return 0, fmt.Errorf("failed to bind local icmp socket: %w", err)
	}
	defer c.Close()

	// 3. Construct a standard ICMP Echo Request message frame
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("NETMON-PING-FRAME"),
		},
	}

	binaryMsg, err := msg.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal icmp packet: %w", err)
	}

	start := time.Now()

	// 4. Set the physical network write deadline to respect the job timeout configuration
	if err := c.SetWriteDeadline(start.Add(timeout)); err != nil {
		return 0, fmt.Errorf("failed to apply socket write deadline: %w", err)
	}

	// Dispatch the packet frame to the target destination
	if _, err := c.WriteTo(binaryMsg, &net.UDPAddr{IP: ip.IP}); err != nil {
		return 0, fmt.Errorf("failed writing packet frame to wire: %w", err)
	}

	// 5. Read the incoming echo response message from the network buffer socket
	replyBuf := make([]byte, 1500)
	if err := c.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return 0, fmt.Errorf("failed to apply socket read deadline: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			n, peer, err := c.ReadFrom(replyBuf)
			if err != nil {
				return 0, fmt.Errorf("ping read timeout or connection drop: %w", err)
			}

			// Validate that the responding node matches our original target destination
			if peer.String() != ip.IP.String() {
				continue
			}

			// Parse the raw network response frame bytes back into structural protocols
			parsedMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), replyBuf[:n])
			if err != nil {
				return 0, fmt.Errorf("corrupted icmp response frame parsed: %w", err)
			}

			switch parsedMsg.Type {
			case ipv4.ICMPTypeEchoReply:
				return time.Since(start), nil
			case ipv4.ICMPTypeDestinationUnreachable:
				return 0, fmt.Errorf("destination target host unreachable network error")
			}
		}
	}
}
