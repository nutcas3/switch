package main

// maskPAN masks a PAN for logging
func maskPAN(pan string) string {
	if len(pan) < 10 {
		return "****"
	}
	return pan[:6] + "****" + pan[len(pan)-4:]
}
