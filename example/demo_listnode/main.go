package main

import (
	"bytes"
	"fmt"
	"math/rand"
)

type ListNode struct {
	Id    int
	State int
	Next  *ListNode
}

func NewListNode(id int) *ListNode {
	l := &ListNode{
		Id:    id,
		State: 0,
		Next:  nil,
	}
	return l
}

func (l *ListNode) Release() {
	l.State = 0
	l.Next = nil
}

func (l *ListNode) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("ListNode(")
	curr := l
	buffer.WriteString(fmt.Sprintf("%d(%d)", curr.Id, curr.State))
	for curr.Next != nil {
		curr = curr.Next
		buffer.WriteString(fmt.Sprintf(" -> %d(%d)", curr.Id, curr.State))
	}
	buffer.WriteString(")")
	return buffer.String()
}

type Manager struct {
	root     *ListNode
	currNode *ListNode
	lastNode *ListNode
}

func NewManager() *Manager {
	m := &Manager{
		root: NewListNode(0),
	}
	rand.Seed(2)
	m.currNode = m.root
	for i := 1; i < 10; i++ {
		m.lastNode = NewListNode(i)
		m.currNode.Next = m.lastNode
		m.currNode = m.lastNode
	}
	fmt.Printf("NewManager | %+v\n", m.root)
	return m
}

func (m *Manager) Run() {
	var random int
	m.currNode = m.root
	for m.currNode != nil {
		random = rand.Intn(100)
		if random < 50 {
			m.currNode.State = 1
		} else {
			m.currNode.State = 0
		}
		m.currNode = m.currNode.Next
	}
	fmt.Printf("(m *Manager) Run | Before Release | %+v\n", m.root)
	m.Release()
	fmt.Printf("(m *Manager) Run | After Release | %+v\n", m.root)
}

func (m *Manager) Release() {
	var preNode *ListNode = nil
	m.currNode = m.root

	for m.currNode != nil {
		if m.currNode.State == 1 {
			if preNode == nil {
				// 更新 root 指標
				m.root = m.currNode.Next

				// 釋放 currNode
				m.currNode.Release()

				// 將 currNode 移到最後一個節點的後面
				m.lastNode.Next = m.currNode

				// 更新最後一個節點的指標
				m.lastNode = m.lastNode.Next

				// 更新 currNode 的指標
				m.currNode = m.root

			} else {
				// 更新 preNode 的子節點
				preNode.Next = m.currNode.Next

				// 釋放 currNode
				m.currNode.Release()

				// 將 currNode 移到最後一個節點的後面
				m.lastNode.Next = m.currNode

				// 更新最後一個節點的指標
				m.lastNode = m.lastNode.Next

				// 更新 currNode 的指標
				m.currNode = preNode.Next
			}
		} else {
			preNode = m.currNode
			m.currNode = m.currNode.Next
		}
	}
}

func main() {
	rand.Seed(2)
	mrg := NewManager()
	mrg.Run()

	// nRound := 10
	// var random int
	// for i := 0; i < nRound; i++ {
	// 	node := root
	// 	for node != nil {
	// 		random = rand.Intn(100)
	// 		if random < 50 {
	// 			node.State = 1
	// 		} else {
	// 			node.State = 0
	// 		}
	// 		node = node.Next
	// 	}

	// 	node = root
	// }
}
