package amount

// CaclulateFee TODO
func CaclulateFee(vinCount int, voutCount int) Amount {
	return Amount(vinCount+voutCount*2) * (COIN / 100)
}
