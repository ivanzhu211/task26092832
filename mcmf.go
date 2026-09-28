package cellprotect

// min-cost max-flow over a transportation network, used to compute tight
// feasible bounds for a single cell without enumerating tables.

type flowEdge struct {
	to, rev, cap, cost int
}

type flowGraph struct {
	g [][]flowEdge
}

func newFlowGraph(n int) *flowGraph {
	return &flowGraph{g: make([][]flowEdge, n)}
}

func (f *flowGraph) addEdge(u, v, cap, cost int) {
	f.g[u] = append(f.g[u], flowEdge{to: v, rev: len(f.g[v]), cap: cap, cost: cost})
	f.g[v] = append(f.g[v], flowEdge{to: u, rev: len(f.g[u]) - 1, cap: 0, cost: -cost})
}

// minCostFlow pushes exactly amount units of flow from s to t and returns
// the minimum cost. The caller guarantees feasibility.
func (f *flowGraph) minCostFlow(s, t, amount int) int {
	const inf = int(^uint(0) >> 2)
	n := len(f.g)
	dist := make([]int, n)
	prevV := make([]int, n)
	prevE := make([]int, n)
	inQueue := make([]bool, n)
	total := 0
	for flow := 0; flow < amount; {
		// Shortest path by residual cost (SPFA; costs may be negative).
		for i := range dist {
			dist[i] = inf
		}
		dist[s] = 0
		queue := []int{s}
		inQueue[s] = true
		for len(queue) > 0 {
			u := queue[0]
			queue = queue[1:]
			inQueue[u] = false
			for ei, e := range f.g[u] {
				if e.cap > 0 && dist[u]+e.cost < dist[e.to] {
					dist[e.to] = dist[u] + e.cost
					prevV[e.to] = u
					prevE[e.to] = ei
					if !inQueue[e.to] {
						queue = append(queue, e.to)
						inQueue[e.to] = true
					}
				}
			}
		}
		if dist[t] == inf {
			break // infeasible; should not happen for valid inputs
		}
		add := amount - flow
		for v := t; v != s; v = prevV[v] {
			if cap := f.g[prevV[v]][prevE[v]].cap; cap < add {
				add = cap
			}
		}
		for v := t; v != s; v = prevV[v] {
			e := &f.g[prevV[v]][prevE[v]]
			e.cap -= add
			f.g[v][e.rev].cap += add
		}
		total += add * dist[t]
		flow += add
	}
	return total
}
