package formstream

// Value first value of the key.
func (p *Parser) Value(key string) (string, Header, bool) {
	value := p.valueMap[key]
	if len(value) == 0 {
		return "", Header{}, false
	}

	content, header := value[0].Unwrap()

	return content, header, true
}

// ValueRaw first value of the key.
func (p *Parser) ValueRaw(key string) ([]byte, Header, bool) {
	value := p.valueMap[key]
	if len(value) == 0 {
		return nil, Header{}, false
	}

	content, header := value[0].UnwrapRaw()

	return content, header, true
}

// Values all values of the key.
func (p *Parser) Values(key string) ([]Value, bool) {
	value, ok := p.valueMap[key]
	if !ok {
		return nil, false
	}

	return value, true
}

// ValueMap all values.
func (p *Parser) ValueMap() map[string][]Value {
	return p.valueMap
}
