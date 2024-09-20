package provider

const defaultOutboxCap uint = 1

var outBoxCap = defaultOutboxCap
var logger Logger = &noopLogger{}
