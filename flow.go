package cellprotect

// Bipartite residual flow network:
//
//	source -> row i (capacity residual row total)
//	row i  -> column j (infinite capacity) for every suppressed cell
//	column j -> sink (capacity residual column total)
//
// The true counts always define a feasible saturating flow.
//
// For a suppressed edge e=(i,j):
//
//	lo(e) = T - maxflow(network with e removed)
//	hi(e) = X - T where X is the maxflow of the network WITH e removed,
//	         augmented with infinite edges i->sink and source->j (forcing
//	         i onto the sink side and j onto the source side of the cut)
//
// Both are exact integer extrema of feasible cell values; no tables are
// enumerated.

const flowInf int64 = 1 << 60

type dinic struct {
	n     int
	graph [][]dinicEdge
	level []int
	ptr   []int
}

type dinicEdge struct {
	to, rev int
	cap     int64
}

func newDinic(n int) *dinic {
	return &dinic{n: n, graph: make([][]dinicEdge, n), level: make([]int, n), ptr: make([]int, n)}
}

func (d *dinic) addEdge(from, to int, cap int64) {
	d.graph[from] = append(d.graph[from], dinicEdge{to: to, rev: len(d.graph[to]), cap: cap})
	d.graph[to] = append(d.graph[to], dinicEdge{to: from, rev: len(d.graph[from]) - 1, cap: 0})
}

func (d *dinic) bfs(s, t int) bool {
	for i := range d.level {
		d.level[i] = -1
	}
	d.level[s] = 0
	queue := []int{s}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		for _, e := range d.graph[v] {
			if e.cap > 0 && d.level[e.to] < 0 {
				d.level[e.to] = d.level[v] + 1
				queue = append(queue, e.to)
			}
		}
	}
	return d.level[t] >= 0
}

func (d *dinic) dfs(v, t int, pushed int64) int64 {
	if v == t || pushed == 0 {
		return pushed
	}
	for ; d.ptr[v] < len(d.graph[v]); d.ptr[v]++ {
		e := &d.graph[v][d.ptr[v]]
		if e.cap > 0 && d.level[e.to] == d.level[v]+1 {
			tr := d.dfs(e.to, t, min64(pushed, e.cap))
			if tr > 0 {
				e.cap -= tr
				d.graph[e.to][e.rev].cap += tr
				return tr
			}
		}
	}
	return 0
}

func (d *dinic) maxFlow(s, t int) int64 {
	var flow int64
	for d.bfs(s, t) {
		for i := range d.ptr {
			d.ptr[i] = 0
		}
		for {
			pushed := d.dfs(s, t, flowInf)
			if pushed == 0 {
				break
			}
			flow += pushed
		}
	}
	return flow
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// flowNetwork stores the current suppression mask in residual form.
type flowNetwork struct {
	rows, cols int
	total      int64
	rRes       []int64
	cRes       []int64
	edges      [][2]int // suppressed cells: (row, col)
}

// buildNetwork constructs the residual network for a suppression mask.
// published[i][j] is true when the cell value is published exactly.
func buildNetwork(c Counts, suppressed [][]bool) flowNetwork {
	m, n := len(c.Cells), 0
	if m > 0 {
		n = len(c.Cells[0])
	}
	net := flowNetwork{
		rows:  m,
		cols:  n,
		rRes:  make([]int64, m),
		cRes:  make([]int64, n),
		edges: make([][2]int, 0, m*n),
	}
	for i := 0; i < m; i++ {
		var rr int64 = int64(c.RowTotals[i])
		for j := 0; j < n; j++ {
			if suppressed[i][j] {
				net.edges = append(net.edges, [2]int{i, j})
			} else {
				rr -= int64(c.Cells[i][j])
			}
		}
		net.rRes[i] = rr
	}
	for j := 0; j < n; j++ {
		var cc int64 = int64(c.ColTotals[j])
		for i := 0; i < m; i++ {
			if !suppressed[i][j] {
				cc -= int64(c.Cells[i][j])
			}
		}
		net.cRes[j] = cc
	}
	for _, rr := range net.rRes {
		net.total += rr
	}
	return net
}

// node ids: source=0, rows 1..m, cols m+1..m+n, sink=m+n+1.
func (net flowNetwork) rowNode(i int) int { return 1 + i }
func (net flowNetwork) colNode(j int) int { return 1 + net.rows + j }
func (net flowNetwork) sink() int         { return 1 + net.rows + net.cols }

// assemble creates a Dinic network, optionally skipping one edge and
// optionally adding the forcing edges for the upper-bound cut.
func (net flowNetwork) assemble(skipIdx int, forceIdx int) *dinic {
	d := newDinic(net.sink() + 1)
	s, t := 0, net.sink()
	for i, r := range net.rRes {
		if r > 0 {
			d.addEdge(s, net.rowNode(i), r)
		}
	}
	for j, cc := range net.cRes {
		if cc > 0 {
			d.addEdge(net.colNode(j), t, cc)
		}
	}
	for idx, e := range net.edges {
		if idx == skipIdx {
			continue
		}
		d.addEdge(net.rowNode(e[0]), net.colNode(e[1]), flowInf)
	}
	if forceIdx >= 0 {
		e := net.edges[forceIdx]
		d.addEdge(net.rowNode(e[0]), t, flowInf)
		d.addEdge(s, net.colNode(e[1]), flowInf)
	}
	return d
}

// edgeRange returns the minimum and maximum feasible value of the
// suppressed cell at edge index idx.
func (net flowNetwork) edgeRange(idx int) (lo, hi int64) {
	if net.total == 0 {
		return 0, 0
	}
	without := net.assemble(idx, -1)
	flow := without.maxFlow(0, net.sink())
	lo = net.total - flow

	forced := net.assemble(idx, idx)
	cutFlow := forced.maxFlow(0, net.sink())
	hi = cutFlow - net.total
	return lo, hi
}
