package main

import "invoice-generator/pkg/invoice"

type Request struct {
	Telephone        string
	BillingStartDate string // AAAA-MM-DD
	BillingEndDate   string // AAAA-MM-DD
	Calls            []invoice.Call
}

// -----------

func main() {

}
