package diff

type Myers struct {
	a []string
	b []string
}

func Diff(a, b []string) []string {
	m := &Myers{
		a: a,
		b: b,
	}
	return m.diff()
}

func (m *Myers) diff() []string {
	diff := []string{}

	m.backtrack(func(prev_x, prev_y, x, y int) {
		if x == prev_x {
			b_line := m.b[prev_y]
			diff = append(diff, NewEdit(INS, b_line).String())
			return
		}

		a_line := m.a[prev_x]
		if y == prev_y {
			diff = append(diff, NewEdit(DEL, a_line).String())
		} else {
			diff = append(diff, NewEdit(EQL, a_line).String())
		}
	})

	for i := 0; i < len(diff)/2; i++ {
		diff[i], diff[len(diff)-i-1] = diff[len(diff)-i-1], diff[i]
	}
	return diff
}

func (s *Myers) backtrack(fn func(int, int, int, int)) {
	x, y := len(s.a), len(s.b)
	max := len(s.a) + len(s.b)

	trace := s.shortestEdit()

	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && v[max+k-1] < v[max+k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[max+prevK]
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			fn(x-1, y-1, x, y)
			x--
			y--
		}

		if d > 0 {
			fn(prevX, prevY, x, y)
			x, y = prevX, prevY
		}
	}
}

func (s *Myers) shortestEdit() [][]int {
	n, m := len(s.a), len(s.b)
	max := n + m
	v := make([]int, 2*max+1)
	v[max] = 0
	var trace [][]int

	for d := 0; d <= max; d++ {
		vCopy := make([]int, len(v))
		copy(vCopy, v)
		trace = append(trace, vCopy)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[max+k-1] < v[max+k+1]) {
				x = v[max+k+1]
			} else {
				x = v[max+k-1] + 1
			}

			y := x - k

			for x < n && y < m && s.a[x] == s.b[y] {
				x, y = x+1, y+1
			}

			v[max+k] = x

			if x >= n && y >= m {
				return trace
			}
		}
	}
	return [][]int{}
}
