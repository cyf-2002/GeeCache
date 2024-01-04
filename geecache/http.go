package geecache

import (
	"fmt"
	"geecache/consistenthash"
	pb "geecache/geecachepb"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTPPool http服务端
type HTTPPool struct {
	self        string // base URL, eg "https://example.net:8000"
	basePath    string // 节点间通讯地址的前缀
	mu          sync.Mutex
	peers       *consistenthash.Map    // 一致性哈希算法的 Map
	httpGetters map[string]*httpGetter // 每一个远程节点对应一个 httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info
func (p *HTTPPool) Log(format string, v ...interface{}) {
	// Sprintf 根据格式指定符进行格式化，并返回结果字符串
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 约定访问路径格式为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		// 服务器内部错误
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 使用 proto.Marshal() 编码 HTTP 响应。
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// Set 添加传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 实现 PeerPicker 接口,根据 key 选择节点
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// http客户端
type httpGetter struct {
	// 要访问的远程节点的地址 eg.http://example.com/_geecache/
	baseURL string
}

// Get 实现 PeerGetter 接口
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	// QueryEscape escapes the string, so it can be safely placed inside a URL query.
	u := fmt.Sprintf("%v%v%v", h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET request failed with status: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	// proto.Unmarshal() 解码 HTTP 响应
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}
