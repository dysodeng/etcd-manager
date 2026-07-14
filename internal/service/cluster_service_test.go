package service

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/dysodeng/etcd-manager/internal/config"
	etcdclient "github.com/dysodeng/etcd-manager/internal/etcd"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"google.golang.org/grpc"
)

type clusterFixtureServer struct {
	etcdserverpb.UnimplementedClusterServer
	etcdserverpb.UnimplementedMaintenanceServer

	memberID uint64
	leaderID uint64
	members  []*etcdserverpb.Member

	statusStarted chan<- uint64
	statusRelease <-chan struct{}
}

func (s *clusterFixtureServer) MemberList(context.Context, *etcdserverpb.MemberListRequest) (*etcdserverpb.MemberListResponse, error) {
	return &etcdserverpb.MemberListResponse{
		Header:  &etcdserverpb.ResponseHeader{ClusterId: 99, MemberId: s.memberID},
		Members: s.members,
	}, nil
}

func (s *clusterFixtureServer) Status(context.Context, *etcdserverpb.StatusRequest) (*etcdserverpb.StatusResponse, error) {
	if s.statusStarted != nil {
		s.statusStarted <- s.memberID
	}
	if s.statusRelease != nil {
		<-s.statusRelease
	}
	return &etcdserverpb.StatusResponse{
		Header:           &etcdserverpb.ResponseHeader{ClusterId: 99, MemberId: s.memberID},
		Version:          "3.5.5",
		DbSize:           int64(s.memberID * 1024),
		DbSizeInUse:      int64(s.memberID * 768),
		Leader:           s.leaderID,
		RaftIndex:        100 + s.memberID,
		RaftTerm:         3,
		RaftAppliedIndex: 100 + s.memberID,
	}, nil
}

func newClusterServiceFixture(t *testing.T) (*ClusterService, []string, []*etcdserverpb.Member, []*clusterFixtureServer) {
	return newClusterServiceFixtureWithConfiguredEndpoints(t, 1)
}

func newClusterServiceFixtureWithConfiguredEndpoints(t *testing.T, configuredEndpointCount int) (*ClusterService, []string, []*etcdserverpb.Member, []*clusterFixtureServer) {
	t.Helper()

	listeners := make([]net.Listener, 3)
	endpoints := make([]string, 3)
	for i := range listeners {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		listeners[i] = listener
		endpoints[i] = "http://" + listener.Addr().String()
	}

	members := make([]*etcdserverpb.Member, 3)
	for i, endpoint := range endpoints {
		memberID := uint64(i + 1)
		members[i] = &etcdserverpb.Member{
			ID:         memberID,
			Name:       fmt.Sprintf("etcd-%d", i),
			PeerURLs:   []string{fmt.Sprintf("http://etcd-%d:2380", i)},
			ClientURLs: []string{endpoint},
		}
	}

	servers := make([]*clusterFixtureServer, len(listeners))
	for i, listener := range listeners {
		server := grpc.NewServer()
		fixture := &clusterFixtureServer{
			memberID: uint64(i + 1),
			leaderID: 1,
			members:  members,
		}
		servers[i] = fixture
		etcdserverpb.RegisterClusterServer(server, fixture)
		etcdserverpb.RegisterMaintenanceServer(server, fixture)
		go func() { _ = server.Serve(listener) }()
		t.Cleanup(server.Stop)
	}

	client, err := etcdclient.NewClient(config.EtcdConfig{Endpoints: endpoints[:configuredEndpointCount]})
	if err != nil {
		t.Fatalf("new etcd client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return NewClusterService(client), endpoints, members, servers
}

func TestClusterServiceMemberStatusesPreferReachableConfiguredEndpoints(t *testing.T) {
	service, endpoints, members, _ := newClusterServiceFixtureWithConfiguredEndpoints(t, 3)
	for i, member := range members {
		member.ClientURLs = []string{fmt.Sprintf("http://127.0.0.1:%d", i+1)}
	}

	statuses, err := service.MemberStatuses(context.Background())
	if err != nil {
		t.Fatalf("member statuses: %v", err)
	}
	if len(statuses) != len(members) {
		t.Fatalf("member status count = %d, want %d", len(statuses), len(members))
	}
	for i, status := range statuses {
		if status.Endpoint != endpoints[i] {
			t.Errorf("status[%d].Endpoint = %q, want reachable configured endpoint %q", i, status.Endpoint, endpoints[i])
		}
	}
}

func TestClusterServiceMemberStatusesDiscoversEveryMember(t *testing.T) {
	service, endpoints, _, _ := newClusterServiceFixture(t)

	statuses, err := service.MemberStatuses(context.Background())
	if err != nil {
		t.Fatalf("member statuses: %v", err)
	}
	if len(statuses) != len(endpoints) {
		t.Fatalf("member status count = %d, want %d", len(statuses), len(endpoints))
	}
	for i, status := range statuses {
		if status.Name != fmt.Sprintf("etcd-%d", i) {
			t.Errorf("status[%d].Name = %q", i, status.Name)
		}
		if status.Endpoint != endpoints[i] {
			t.Errorf("status[%d].Endpoint = %q, want %q", i, status.Endpoint, endpoints[i])
		}
	}
}

func TestClusterServiceMetricsChecksEveryMember(t *testing.T) {
	service, endpoints, _, _ := newClusterServiceFixture(t)

	metrics, err := service.Metrics(context.Background())
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	if len(metrics.Health) != len(endpoints) {
		t.Fatalf("health count = %d, want %d", len(metrics.Health), len(endpoints))
	}
	for _, endpoint := range endpoints {
		if !metrics.Health[endpoint] {
			t.Errorf("endpoint %q was not reported healthy", endpoint)
		}
	}
}

func TestClusterServiceMemberStatusesPrefersUniqueMemberURLs(t *testing.T) {
	service, endpoints, members, _ := newClusterServiceFixture(t)
	sharedEndpoint := endpoints[1]
	members[0].ClientURLs = []string{sharedEndpoint, endpoints[0]}
	members[1].ClientURLs = []string{sharedEndpoint}

	statuses, err := service.MemberStatuses(context.Background())
	if err != nil {
		t.Fatalf("member statuses: %v", err)
	}
	if len(statuses) != len(members) {
		t.Fatalf("member status count = %d, want %d", len(statuses), len(members))
	}
	seen := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		if seen[status.Name] {
			t.Fatalf("member %q appeared more than once", status.Name)
		}
		seen[status.Name] = true
	}
	for _, member := range members {
		if !seen[member.Name] {
			t.Errorf("member %q is missing", member.Name)
		}
	}
}

func TestClusterServiceMemberStatusesProbesMembersConcurrently(t *testing.T) {
	service, _, _, servers := newClusterServiceFixtureWithConfiguredEndpoints(t, 3)
	started := make(chan uint64, len(servers))
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	for _, server := range servers {
		server.statusStarted = started
		server.statusRelease = release
	}

	result := make(chan error, 1)
	go func() {
		statuses, err := service.MemberStatuses(context.Background())
		if err == nil && len(statuses) != len(servers) {
			err = fmt.Errorf("member status count = %d, want %d", len(statuses), len(servers))
		}
		result <- err
	}()

	seen := make(map[uint64]bool, len(servers))
	deadline := time.After(time.Second)
	for len(seen) < len(servers) {
		select {
		case memberID := <-started:
			seen[memberID] = true
		case <-deadline:
			t.Fatalf("only %d/%d member probes started before the first completed", len(seen), len(servers))
		}
	}
	close(release)
	released = true
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
