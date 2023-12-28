创建一个结构体 `HTTPPool`，作为承载节点间 HTTP 通信的核心数据结构（包括服务端和客户端，今天只实现服务端）

ServeHTTP 的实现逻辑是比较简单的，首先判断访问路径的前缀是否是 `basePath`，不是返回错误。约定访问路径格式为 `/<basepath>/<groupname>/<key>`，通过 groupname 得到 group 实例，再使用 `group.Get(key)` 获取缓存数据。

```go
package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_geecache/"

type HTTPPool struct {
	self     string // base URL, eg "https://example.net:8000"
	basePath string // 节点间通讯地址的前缀
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
		http.Error(w, "no such group: " + groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		// 服务器内部错误
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
```

