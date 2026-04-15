package skill

// MergeRegistries merges b into a. Same `name` is overridden by the pack from b (custom skills override built-ins).
func MergeRegistries(a, b *Registry) *Registry {
	if a == nil && (b == nil || len(b.Packs) == 0) {
		return &Registry{}
	}
	if b == nil || len(b.Packs) == 0 {
		return a
	}
	if a == nil || len(a.Packs) == 0 {
		return b
	}
	byName := make(map[string]Pack)
	var order []string
	seen := map[string]struct{}{}
	for _, p := range a.Packs {
		if p.Name == "" {
			continue
		}
		byName[p.Name] = p
		if _, ok := seen[p.Name]; !ok {
			seen[p.Name] = struct{}{}
			order = append(order, p.Name)
		}
	}
	for _, p := range b.Packs {
		if p.Name == "" {
			continue
		}
		byName[p.Name] = p
		if _, ok := seen[p.Name]; !ok {
			seen[p.Name] = struct{}{}
			order = append(order, p.Name)
		}
	}
	out := make([]Pack, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return &Registry{Packs: out}
}
