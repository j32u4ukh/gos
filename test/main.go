package main

import (
	"fmt"
	"strconv"
	"strings"
)

type EndPoint struct {
	nodes    []*node
	nNode    int32
	priority int32
	params   map[string]any
}

func NewEndPoint() *EndPoint {
	ep := &EndPoint{
		nodes:    []*node{},
		nNode:    0,
		priority: 0,
		params:   make(map[string]any),
	}
	return ep
}

func (ep *EndPoint) InitNodes(url string) error {
	splits := strings.Split(url, "/")
	nSplit := len(splits)
	fmt.Printf("splits(#%d): %+v\n", nSplit, splits)
	var n *node
	for _, s := range splits {
		n = newNode(s)
		if !n.isParam {
			ep.priority += 1
		}
		ep.nodes = append(ep.nodes, n)
	}
	ep.nNode = int32(len(ep.nodes))
	return nil
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
	fmt.Printf("params: %+v\n", ep.params)
	return true
}

func (ep *EndPoint) SetParam(key string, value any) {
	ep.params[key] = value
}

type node struct {
	route     string
	routeType string
	isParam   bool
	value     any
}

func newNode(route string) *node {
	n := new(node)
	if strings.HasPrefix(route, "<") && strings.HasSuffix(route, ">") {
		n.isParam = true
		route = route[1 : len(route)-1]
	}
	routes := strings.Split(route, " ")
	n.route = routes[0]
	if len(routes) > 1 {
		n.routeType = routes[1]
	}
	return n
}

func (n *node) match(route string) bool {
	if n.isParam {
		switch n.routeType {
		case "int":
			i, err := strconv.Atoi(route)
			if err != nil {
				return false
			}
			n.value = i
		case "float":
			f, err := strconv.ParseFloat(route, 64)
			if err != nil {
				return false
			}
			n.value = f
		case "string", "":
			n.value = route
		default:
			return false
		}
		return true
	} else {
		return route == n.route
	}
}

func main() {
	ep1 := NewEndPoint()
	ep1.InitNodes("/post/<name>/456")

	ep2 := NewEndPoint()
	// ep2.InitNodes("/post/<user_id>/<post_id>")
	ep2.InitNodes("/post/<user_name>/<post_id int>")
	// ep2.InitNodes("/post/<name>/<tag>")

	ep3 := NewEndPoint()
	ep3.InitNodes("/")

	fmt.Printf("ep1: %d, ep2: %d, ep3: %d\n", ep1.priority, ep2.priority, ep3.priority)

	url := "/post/user_id_a9527/123"
	splits := strings.Split(url, "/")
	fmt.Printf("match ep1? %v\n", ep1.Macth(splits))
	fmt.Printf("match ep2? %v\n", ep2.Macth(splits))

	// url = "/"
	// splits = strings.Split(url, "/")
	// fmt.Printf("match ep1? %v\n", ep1.Macth(splits))
	// fmt.Printf("match ep2? %v\n", ep2.Macth(splits))
	// fmt.Printf("match ep3? %v\n", ep3.Macth(splits))
}
