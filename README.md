# nomad-yamux-model

_A model application demonstrating Nomad's client failover behavior, as well as
bpftrace programs that trace real Nomad client failover behavior._

This repo is a companion to https://github.com/hashicorp/nomad/issues/14869. The bpftrace code here has only been tested with Nomad 1.4.1, and because it relies on reading registers figured out from the disassembled binary, they are unlikely to work on any other version of Nomad. Your mileage may vary.
