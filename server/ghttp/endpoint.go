package ghttp

type EndPoint struct {
	path     string
	nodes    []*node
	nNode    int32
	priority float32
	params   map[string]any
	// key: HttpMethod(GET/POST/...), value: handler functions
	Handlers map[string]HandlerChain
	options  []string
}

func NewEndPoint() *EndPoint {
	ep := &EndPoint{
		nodes:    []*node{},
		nNode:    0,
		priority: 0,
		params:   make(map[string]any),
		Handlers: map[string]HandlerChain{
			METHOD_OPTIONS: {},
		},
		options: []string{METHOD_OPTIONS},
	}
	return ep
}

func (ep *EndPoint) InitNodes(nodes []*node) {
	var n *node
	for _, n = range nodes {
		if n.isParam {
			if n.routeType == "int" || n.routeType == "uint" || n.routeType == "float" {
				ep.priority += 0.5
			}
		} else {
			ep.priority += 1.0
		}
		ep.nodes = append(ep.nodes, n)
	}
	ep.nNode = int32(len(ep.nodes))
}

func (ep *EndPoint) Macth(routes []string) bool {
	if ep.nNode != int32(len(routes)) {
		return false
	}
	var n *node
	for i, route := range routes {
		n = ep.nodes[i]
		if !n.match(route) {
			return false
		}
	}
	for _, n := range ep.nodes {
		if n.isParam {
			ep.SetParam(n.route, n.value)
		}
	}
	return true
}

func (ep *EndPoint) SetParam(key string, value any) {
	ep.params[key] = value
}
