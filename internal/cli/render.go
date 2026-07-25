// render.go: presentation only - one render function per result type
// (PartitionLayout, InitResult, Plan, …), each taking the io.Writer it
// prints to and returning write errors. Renders whatever data the app
// returned; no decisions, no queries, no I/O beyond the writer.
package cli
