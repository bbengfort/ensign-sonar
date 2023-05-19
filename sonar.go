package sonar

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/rotationalio/go-ensign"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	mimetype "github.com/rotationalio/go-ensign/mimetype/v1beta1"
	"github.com/vmihailenco/msgpack"
)

const (
	Mimetype   = "application/msgpack"
	DefaultTTL = 750 * time.Millisecond
	SchemaName = "ping"
)

type Ping struct {
	Sequence  uint64        `msgpack:"sequence"`
	Hostname  string        `msgpack:"hostname"`
	IPAddress string        `msgpack:"ipaddr"`
	TTL       time.Duration `msgpack:"ttl"`
	Timestamp time.Time     `msgpack:"timestamp"`
	NBytes    int           `msgpack:"-"`
	Received  time.Time     `msgpack:"-"`
}

type Sonar struct {
	sequence uint64
	template Ping
}

func New() *Sonar {
	return &Sonar{
		template: Ping{
			Hostname:  Hostname(),
			IPAddress: GetOutboundIP().String(),
			TTL:       DefaultTTL,
		},
	}
}

func (s *Sonar) Next() *Ping {
	s.sequence++
	return &Ping{
		Sequence:  s.sequence,
		Hostname:  s.template.Hostname,
		IPAddress: s.template.IPAddress,
		TTL:       s.template.TTL,
		Timestamp: time.Now().Truncate(0),
	}
}

func (p *Ping) Marshal() ([]byte, error) {
	return msgpack.Marshal(p)
}

func (p *Ping) Unmarshal(data []byte) error {
	p.Received = time.Now()
	p.NBytes = len(data)
	return msgpack.Unmarshal(data, p)
}

func (p *Ping) Event() *ensign.Event {
	event := &ensign.Event{
		Mimetype: mimetype.MustParse(Mimetype),
		Type: &api.Type{
			Name:         SchemaName,
			MajorVersion: VersionMajor,
			MinorVersion: VersionMinor,
			PatchVersion: VersionPatch,
		},
		Created: time.Now(),
	}

	event.Data, _ = p.Marshal()
	return event
}

func (p *Ping) String() string {
	var sender string
	switch {
	case p.Hostname != "" && p.IPAddress != "":
		sender = fmt.Sprintf("%s (%s)", p.Hostname, p.IPAddress)
	case p.Hostname != "":
		sender = p.Hostname
	case p.IPAddress != "":
		sender = p.IPAddress
	default:
		sender = "unknown"
	}

	return fmt.Sprintf("%d bytes from %s: seq=%d ttl=%s time=%s", p.Size(), sender, p.Sequence, p.TTL, p.Timedelta())
}

func (p *Ping) Size() int {
	if p.NBytes == 0 {
		data, _ := p.Marshal()
		p.NBytes = len(data)
	}
	return p.NBytes
}

func (p *Ping) Timedelta() time.Duration {
	if p.Received.IsZero() {
		p.Received = time.Now()
	}
	return p.Received.Sub(p.Timestamp)
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func Hostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
