package jsonobj

// Equal は構造的な等価判定（vitest の toEqual）。
//
//   - 数値は表現によらず値で比べる（json.Number "1" == int 1 == float64 1）
//   - Undefined の値を持つキーは無いキーと同じに扱う
//   - 配列は要素ごと、オブジェクトはキー集合と各値
//   - []string / []*Object は []any と同じに扱う
func Equal(a, b any) bool {
	if IsUndefined(a) && IsUndefined(b) {
		return true
	}
	if IsUndefined(a) || IsUndefined(b) {
		return false
	}
	if a == nil || b == nil {
		return isNil(a) && isNil(b)
	}
	if fa, ok := ToFloat(a); ok {
		fb, ok := ToFloat(b)
		return ok && fa == fb
	}
	switch x := a.(type) {
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case *Object:
		y, ok := b.(*Object)
		if !ok {
			return false
		}
		return equalObjects(x, y)
	}
	if aa, ok := AsArray(a); ok {
		ba, ok := AsArray(b)
		if !ok || len(aa) != len(ba) {
			return false
		}
		for i := range aa {
			if !Equal(aa[i], ba[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	if o, ok := v.(*Object); ok {
		return o == nil
	}
	return false
}

func equalObjects(x, y *Object) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	for _, k := range x.keys {
		xv := x.vals[k]
		yv, ok := y.vals[k]
		if IsUndefined(xv) {
			if ok && !IsUndefined(yv) {
				return false
			}
			continue
		}
		if !ok || !Equal(xv, yv) {
			return false
		}
	}
	for _, k := range y.keys {
		if IsUndefined(y.vals[k]) {
			continue
		}
		if _, ok := x.vals[k]; !ok {
			return false
		}
	}
	return true
}

// Contains は「a が b のキーと値を全部含む」（vitest の toMatchObject の 1 段版。再帰する）。
func Contains(a, b any) bool {
	ao, ok1 := a.(*Object)
	bo, ok2 := b.(*Object)
	if ok1 && ok2 {
		for _, k := range bo.keys {
			if IsUndefined(bo.vals[k]) {
				continue
			}
			av, ok := ao.vals[k]
			if !ok || !Contains(av, bo.vals[k]) {
				return false
			}
		}
		return true
	}
	return Equal(a, b)
}
