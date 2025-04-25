package ghttp

import (
	"strconv"
	"strings"
)

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
			i, err := strconv.ParseInt(route, 10, 64)
			if err != nil {
				return false
			}
			n.value = i
		case "uint":
			i, err := strconv.ParseUint(route, 10, 64)
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
