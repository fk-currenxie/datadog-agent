package util

import (
	"github.com/DataDog/agent-payload/process"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testSourceFilters = map[string][]string{
	"172.0.0.1":     {"80", "10", "443"},
	"*":             {"9000"}, // only port 9000
	"::7f00:35:0:0": {"443"},  // ipv6
	"10.0.0.10":     {"3333", "*"},
	"10.0.0.25":     {"30", "ABCD"}, // invalid config
	"123.ABCD":      {"*"},          // invalid config

	"172.0.0.2":     {"80", "10", "443", "53361-53370", "100-100"},
	"::7f00:35:0:1": {"65536"}, // invalid port
	"10.0.0.11":     {"3333", "*", "53361-53370"},
	"10.0.0.26":     {"30", "53361-53360"}, // invalid port range
	"10.0.0.1":      {"tcp *", "53361-53370"},
	"10.0.0.2":      {"tcp 53361-53500", "udp 119"},
}

var testDestinationFilters = map[string][]string{
	"10.0.0.0/24":      {"8080", "8081", "10255"},
	"":                 {"1234"}, // invalid config
	"2001:db8::2:1":    {"5001"},
	"2001:db8::2:1/55": {"80"},
	"*":                {"*"}, // invalid config

	"2001:db8::2:2":    {"3333 udp"}, // /invalid config
	"10.0.0.3/24":      {"30-ABC"},   // invalid port range
	"10.0.0.4":         {"udp *", "*"},
	"2001:db8::2:2/55": {"8080-8082-8085"}, // invalid config
}

func TestParseConnectionFilters(t *testing.T) {
	sourceList := ParseConnectionFilters(testSourceFilters)
	destList := ParseConnectionFilters(testDestinationFilters)

	// source
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("172.0.0.1"), uint16(10), process.ConnectionType_tcp))
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("*"), uint16(9000), process.ConnectionType_tcp)) // only port 9000
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.1.24"), uint16(9000), process.ConnectionType_tcp))
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("::7f00:35:0:0"), uint16(443), process.ConnectionType_tcp)) // ipv6
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("0"), uint16(443), process.ConnectionType_tcp))             // 0 == ::7f00:35:0:0
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.10"), uint16(6666), process.ConnectionType_tcp))    // port wildcard
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.10"), uint16(33), process.ConnectionType_tcp))
	assert.False(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.25"), uint16(30), process.ConnectionType_tcp)) // bad config
	assert.False(t, IsBlacklistedConnection(sourceList, AddressFromString("123.ABCD"), uint16(30), process.ConnectionType_tcp))  // bad IP config

	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("172.0.0.2"), uint16(100), process.ConnectionType_tcp))
	assert.False(t, IsBlacklistedConnection(sourceList, AddressFromString("::7f00:35:0:1"), uint16(100), process.ConnectionType_tcp)) // invalid port
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.11"), uint16(6666), process.ConnectionType_tcp))     // port wildcard
	assert.False(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.26"), uint16(30), process.ConnectionType_tcp))      // invalid port range
	assert.True(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.1"), uint16(100), process.ConnectionType_tcp))       // tcp wildcard
	assert.False(t, IsBlacklistedConnection(sourceList, AddressFromString("10.0.0.2"), uint16(53363), process.ConnectionType_udp))    // udp not blacklisted

	// destination
	assert.True(t, IsBlacklistedConnection(destList, AddressFromString("10.0.0.5"), uint16(8080), process.ConnectionType_tcp))
	assert.False(t, IsBlacklistedConnection(destList, AddressFromString("10.0.0.5"), uint16(80), process.ConnectionType_tcp))
	assert.False(t, IsBlacklistedConnection(destList, AddressFromString(""), uint16(1234), process.ConnectionType_tcp))
	assert.True(t, IsBlacklistedConnection(destList, AddressFromString("2001:db8::2:1"), uint16(5001), process.ConnectionType_tcp)) // ipv6
	assert.True(t, IsBlacklistedConnection(destList, AddressFromString("2001:db8::5:1"), uint16(80), process.ConnectionType_tcp))   // ipv6 CIDR
	assert.False(t, IsBlacklistedConnection(destList, AddressFromString("*"), uint16(30), process.ConnectionType_tcp))

	assert.False(t, IsBlacklistedConnection(destList, AddressFromString("2001:db8::2:2"), uint16(3333), process.ConnectionType_udp)) // invalid config
	assert.False(t, IsBlacklistedConnection(destList, AddressFromString("10.0.0.3/24"), uint16(80), process.ConnectionType_tcp))     // invalid config
	assert.True(t, IsBlacklistedConnection(destList, AddressFromString("10.0.0.4"), uint16(1234), process.ConnectionType_tcp))       // port wildcard
	assert.False(t, IsBlacklistedConnection(destList, AddressFromString("2001:db8::2:2"), uint16(8082), process.ConnectionType_tcp)) // invalid config

}

var sink bool

func BenchmarkIsBlacklistedConnectionIPv4(b *testing.B) {
	sourceList := ParseConnectionFilters(testSourceFilters)
	destList := ParseConnectionFilters(testDestinationFilters)
	addrs := randIPv4(6)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, addr := range addrs {
			sink = IsBlacklistedConnection(sourceList, addr, uint16(rand.Intn(9999)), process.ConnectionType_tcp)
			sink = IsBlacklistedConnection(destList, addr, uint16(rand.Intn(9999)), process.ConnectionType_tcp)
		}
	}

}

func BenchmarkIsBlacklistedConnectionIPv6(b *testing.B) {
	sourceList := ParseConnectionFilters(testSourceFilters)
	destList := ParseConnectionFilters(testDestinationFilters)
	addrs := randIPv6(6)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, addr := range addrs {
			sink = IsBlacklistedConnection(sourceList, addr, uint16(rand.Intn(9999)), process.ConnectionType_tcp)
			sink = IsBlacklistedConnection(destList, addr, uint16(rand.Intn(9999)), process.ConnectionType_tcp)
		}
	}
}

func randIPv4(count int) (addrs []Address) {
	for i := 0; i < count; i++ {
		r := rand.Intn(999999999) + 999999
		addrs = append(addrs, V4Address(uint32(r)))
	}
	return addrs
}

func randIPv6(count int) (addrs []Address) {
	for i := 0; i < count; i++ {
		r := rand.Intn(999999999) + 999999
		addrs = append(addrs, V6Address(uint64(r), uint64(r)))
	}
	return addrs
}
