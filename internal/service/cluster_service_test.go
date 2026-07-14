package service

import (
	"context"
	"fmt"
	"net"
	"testing"

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
}

func (s *clusterFixtureServer) MemberList(context.Context, *etcdserverpb.MemberListRequest) (*etcdserverpb.MemberListResponse, error) {
	return &etcdserverpb.MemberListResponse{
		Header:  &etcdserverpb.ResponseHeader{ClusterId: 99, MemberId: s.memberID},
		Members: s.members,
	}, nil
}

func (s *clusterFixtureServer) Status(context.Context, *etcdserverpb.StatusRequest) (*etcdserverpb.StatusResponse, error) {
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

func newClusterServiceFixture(t *testing.T) (*ClusterService, []string) {
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

	for i, listener := range listeners {
		server := grpc.NewServer()
		fixture := &clusterFixtureServer{
			memberID: uint64(i + 1),
			leaderID: 1,
			members:  members,
		}
		etcdserverpb.RegisterClusterServer(server, fixture)
		etcdserverpb.RegisterMaintenanceServer(server, fixture)
		go func() { _ = server.Serve(listener) }()
		t.Cleanup(server.Stop)
	}

	client, err := etcdclient.NewClient(config.EtcdConfig{Endpoints: []string{endpoints[0]}})
	if err != nil {
		t.Fatalf("new etcd client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return NewClusterService(client), endpoints
}

func TestClusterServiceMemberStatusesDiscoversEveryMember(t *testing.T) {
	service, endpoints := newClusterServiceFixture(t)

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
	service, endpoints := newClusterServiceFixture(t)

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
