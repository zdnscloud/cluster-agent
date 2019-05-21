package service

type StringSet map[string]struct{}

func NewStringSet() StringSet {
	return make(StringSet)
}

func (ss StringSet) Member(s string) bool {
	_, ok := ss[s]
	return ok
}

func (ss StringSet) Add(s string) {
	ss[s] = struct{}{}
}

func (ss StringSet) Delete(s string) {
	delete(ss, s)
}

func (ss StringSet) Difference(other StringSet) StringSet {
	res := NewStringSet()
	for s := range ss {
		if _, ok := other[s]; ok == false {
			res.Add(s)
		}
	}
	return res
}
