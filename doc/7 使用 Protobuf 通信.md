将 HTTP 通信的中间载体替换成了 protobuf。运行 `run.sh` 即可以测试 GeeCache 能否正常工作。

## 使用 protobuf 通信

新建 package `geecachepb`，定义 `geecachepb.proto`

```go
syntax = "proto3";

package geecachepb;

message Request {
  string group = 1;
  string key = 2;
}

message Response {
  bytes value = 1;
}

service GroupCache {
  rpc Get(Request) returns (Response);
}
```

修改 `peers.go` 中的 `PeerGetter` 接口，参数使用 `geecachepb.pb.go` 中的数据类型

```go
import pb "geecache/geecachepb"

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
```

最后，修改 `geecache.go` 和 `http.go` 中使用了 `PeerGetter` 接口的地方。

geecache.go

```go
import (
    // ...
    pb "geecache/geecachepb"
)

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
```

http.go

```go
import (
    // ...
	pb "geecache/geecachepb"
	"github.com/golang/protobuf/proto"
)

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // ...
	// Write the value to the response body as a proto message.
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
    res, err := http.Get(u)
	// ...
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}
```