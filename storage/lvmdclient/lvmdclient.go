package lvmdclient

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	lvmd "github.com/google/lvmd/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type LVMConnection interface {
	CreateVG(ctx context.Context, opt *VGOptions) (string, error)
	CreateLV(ctx context.Context, opt *LVOptions) (string, error)

	RemoveVG(ctx context.Context, opt *VGOptions) (string, error)
	RemoveLV(ctx context.Context, vgName string, lvName string) (string, error)

	GetVG(ctx context.Context) ([]*lvmd.VolumeGroup, error)
	GetLV(ctx context.Context, vgName string) ([]*lvmd.LogicalVolume, error)

	Close() error
}

type lvmConnection struct {
	conn *grpc.ClientConn
}

var (
	_ LVMConnection = &lvmConnection{}
)

func NewLVMConnection(address string, timeout time.Duration) (LVMConnection, error) {
	conn, err := connect(address, timeout)
	if err != nil {
		return nil, err
	}
	return &lvmConnection{
		conn: conn,
	}, nil
}

func (c *lvmConnection) Close() error {
	return c.conn.Close()
}

func connect(address string, timeout time.Duration) (*grpc.ClientConn, error) {
	glog.V(2).Infof("Connecting to %s", address)
	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(time.Second),
		grpc.WithUnaryInterceptor(logGRPC),
	}
	if strings.HasPrefix(address, "/") {
		dialOptions = append(dialOptions, grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	}
	conn, err := grpc.Dial(address, dialOptions...)

	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		if !conn.WaitForStateChange(ctx, conn.GetState()) {
			glog.V(4).Infof("Connection timed out")
			return conn, nil // return nil, subsequent GetPluginInfo will show the real connection error
		}
		if conn.GetState() == connectivity.Ready {
			glog.V(3).Infof("Connected")
			return conn, nil
		}
		glog.V(4).Infof("Still trying, connection is %s", conn.GetState())
	}
}

func logGRPC(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	glog.V(5).Infof("GRPC call: %s", method)
	glog.V(5).Infof("GRPC request: %+v", req)
	err := invoker(ctx, method, req, reply, cc, opts...)
	glog.V(5).Infof("GRPC response: %+v", reply)
	glog.V(5).Infof("GRPC error: %v", err)
	return err
}

type LVOptions struct {
	VolumeGroup string
	Name        string
	Size        uint64
}

type VGOptions struct {
	Name           string
	PhysicalVolume string
}

func (c *lvmConnection) CreateLV(ctx context.Context, opt *LVOptions) (string, error) {
	client := lvmd.NewLVMClient(c.conn)

	req := lvmd.CreateLVRequest{
		VolumeGroup: opt.VolumeGroup,
		Name:        opt.Name,
		Size:        opt.Size,
	}

	rsp, err := client.CreateLV(ctx, &req)
	if err != nil {
		return "", err
	}
	return rsp.GetCommandOutput(), nil
}

func (c *lvmConnection) RemoveLV(ctx context.Context, vgName string, lvNmae string) (string, error) {
	client := lvmd.NewLVMClient(c.conn)

	req := lvmd.RemoveLVRequest{
		VolumeGroup: vgName,
		Name:        lvNmae,
	}

	rsp, err := client.RemoveLV(ctx, &req)
	glog.V(5).Infof("removeLV output: %v", rsp.GetCommandOutput())
	return rsp.GetCommandOutput(), err
}

func (c *lvmConnection) GetLV(ctx context.Context, vgName string) ([]*lvmd.LogicalVolume, error) {
	client := lvmd.NewLVMClient(c.conn)

	req := lvmd.ListLVRequest{
		VolumeGroup: fmt.Sprintf("%s", vgName),
	}

	rsp, err := client.ListLV(ctx, &req)

	if err != nil {
		return nil, err
	}
	return rsp.GetVolumes(), nil
	return nil, err
}

func (c *lvmConnection) CreateVG(ctx context.Context, opt *VGOptions) (string, error) {
	client := lvmd.NewLVMClient(c.conn)
	req := lvmd.CreateVGRequest{
		Name:           opt.Name,
		PhysicalVolume: opt.PhysicalVolume,
	}
	rsp, err := client.CreateVG(ctx, &req)
	if err != nil {
		return "", err
	}
	return rsp.GetCommandOutput(), nil
}

func (c *lvmConnection) GetVG(ctx context.Context) ([]*lvmd.VolumeGroup, error) {
	client := lvmd.NewLVMClient(c.conn)

	req := lvmd.ListVGRequest{}
	rsp, err := client.ListVG(ctx, &req)

	if err != nil {
		return nil, err
	}
	return rsp.GetVolumeGroups(), nil

	return nil, err
}

func (c *lvmConnection) RemoveVG(ctx context.Context, opt *VGOptions) (string, error) {
	client := lvmd.NewLVMClient(c.conn)

	req := lvmd.CreateVGRequest{
		Name:           opt.Name,
		PhysicalVolume: opt.PhysicalVolume,
	}

	rsp, err := client.RemoveVG(ctx, &req)
	glog.V(5).Infof("removeVG output: %v", rsp.GetCommandOutput())
	return rsp.GetCommandOutput(), err
}
