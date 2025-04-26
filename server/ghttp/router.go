package ghttp

import (
	"path"
	"sort"
	"strings"
)

type Router struct {
	path      string
	nodes     []*node
	Handlers  HandlerChain
	endpoints *EndPointHandlers
}

// 每個 EndPoint 對應一個 Router，但每個 Router 不一定對應著一個 EndPoint
func (r *Router) NewRouter(relativePath string, handlers ...HandlerFunc) *Router {
	nr := &Router{
		endpoints: r.endpoints,
		path:      r.combinePath(relativePath),
		nodes:     r.combineNodes(relativePath),
		Handlers:  r.combineHandlers(handlers),
	}
	return nr
}

func (r *Router) HEAD(path string, handlers ...HandlerFunc) {
	r.handle(METHOD_HEAD, path, handlers...)
}

func (r *Router) GET(path string, handlers ...HandlerFunc) {
	r.handle(METHOD_GET, path, handlers...)
}

func (r *Router) POST(path string, handlers ...HandlerFunc) {
	r.handle(METHOD_POST, path, handlers...)
}

func (r *Router) PUT(path string, handlers ...HandlerFunc) {
	r.handle(METHOD_PUT, path, handlers...)
}

func (r *Router) PATCH(path string, handlers ...HandlerFunc) {
	r.handle(METHOD_PATCH, path, handlers...)
}

func (r *Router) DELETE(path string, handlers ...HandlerFunc) {
	r.handle(METHOD_DELETE, path, handlers...)
}

func (r *Router) handle(method string, path string, handlers ...HandlerFunc) {
	var endpoint *EndPoint
	fullPath := r.combinePath(path)
	isExists := false
	for _, endpoint = range *r.endpoints {
		if endpoint.path == fullPath {
			isExists = true
			break
		}
	}
	if !isExists {
		endpoint = NewEndPoint()
		endpoint.path = fullPath
		nodes := r.combineNodes(path)
		endpoint.InitNodes(nodes)
		*r.endpoints = append(*r.endpoints, endpoint)
	}
	if _, ok := endpoint.Handlers[method]; !ok {
		endpoint.options = append(endpoint.options, method)
	}
	endpoint.Handlers[method] = r.combineHandlers(handlers)
	sort.SliceStable(*r.endpoints, func(i, j int) bool {
		// True 的話，會被排到前面
		return (*r.endpoints)[i].priority > (*r.endpoints)[j].priority
	})
}

func (r *Router) combinePath(relativePath string) string {
	return strings.TrimRight(path.Join(r.path, relativePath), "/")
}

func (r *Router) combineNodes(relativePath string) []*node {
	nodes := []*node{}
	nodes = append(nodes, r.nodes...)
	splits := strings.Split(relativePath, "/")
	var n *node
	for _, s := range splits {
		if s == "" {
			continue
		}
		n = newNode(s)
		nodes = append(nodes, n)
	}
	return nodes
}

func (r *Router) combineHandlers(handlers HandlerChain) HandlerChain {
	size := len(r.Handlers) + len(handlers)
	mergedHandlers := make(HandlerChain, size)
	copy(mergedHandlers, r.Handlers)
	copy(mergedHandlers[len(r.Handlers):], handlers)
	return mergedHandlers
}
