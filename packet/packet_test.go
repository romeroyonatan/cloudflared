package packet

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestNewICMPTTLExceedPacket(t *testing.T) {
	ipv4Packet := IP{
		Src:      netip.MustParseAddr("192.168.1.1"),
		Dst:      netip.MustParseAddr("10.0.0.1"),
		Protocol: layers.IPProtocolICMPv4,
		TTL:      0,
	}
	icmpV4Packet := ICMP{
		IP: &ipv4Packet,
		Message: &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   25821,
				Seq:  58129,
				Data: []byte("test ttl=0"),
			},
		},
	}
	assertTTLExceedPacket(t, &icmpV4Packet)
	icmpV4Packet.Body = &icmp.Echo{
		ID:   3487,
		Seq:  19183,
		Data: make([]byte, ipv4MinMTU),
	}
	assertTTLExceedPacket(t, &icmpV4Packet)
	ipv6Packet := IP{
		Src:      netip.MustParseAddr("fd51:2391:523:f4ee::1"),
		Dst:      netip.MustParseAddr("fd51:2391:697:f4ee::2"),
		Protocol: layers.IPProtocolICMPv6,
		TTL:      0,
	}
	icmpV6Packet := ICMP{
		IP: &ipv6Packet,
		Message: &icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   25821,
				Seq:  58129,
				Data: []byte("test ttl=0"),
			},
		},
	}
	assertTTLExceedPacket(t, &icmpV6Packet)
	icmpV6Packet.Body = &icmp.Echo{
		ID:   1497,
		Seq:  39284,
		Data: make([]byte, ipv6MinMTU),
	}
	assertTTLExceedPacket(t, &icmpV6Packet)
}

func assertTTLExceedPacket(t *testing.T, pk *ICMP) {
	encoder := NewEncoder()
	rawPacket, err := encoder.Encode(pk)
	require.NoError(t, err)

	minMTU := ipv4MinMTU
	headerLen := ipv4MinHeaderLen
	routerIP := netip.MustParseAddr("172.16.0.3")
	if pk.Dst.Is6() {
		minMTU = ipv6MinMTU
		headerLen = ipv6HeaderLen
		routerIP = netip.MustParseAddr("fd51:2391:697:f4ee::3")
	}

	ttlExceedPacket := NewICMPTTLExceedPacket(pk.IP, rawPacket, routerIP)
	require.Equal(t, routerIP, ttlExceedPacket.Src)
	require.Equal(t, pk.Src, ttlExceedPacket.Dst)
	require.Equal(t, pk.Protocol, ttlExceedPacket.Protocol)
	require.Equal(t, DefaultTTL, ttlExceedPacket.TTL)

	timeExceed, ok := ttlExceedPacket.Body.(*icmp.TimeExceeded)
	require.True(t, ok)
	if len(rawPacket.Data) > minMTU {
		require.True(t, bytes.Equal(timeExceed.Data, rawPacket.Data[:minMTU-headerLen-icmpHeaderLen]))
	} else {
		require.True(t, bytes.Equal(timeExceed.Data, rawPacket.Data))
	}

	rawTTLExceedPacket, err := encoder.Encode(ttlExceedPacket)
	require.NoError(t, err)
	if len(rawPacket.Data) > minMTU {
		require.Len(t, rawTTLExceedPacket.Data, minMTU)
	} else {
		require.Len(t, rawTTLExceedPacket.Data, headerLen+icmpHeaderLen+len(rawPacket.Data))
		require.True(t, bytes.Equal(rawPacket.Data, rawTTLExceedPacket.Data[headerLen+icmpHeaderLen:]))
	}
}
