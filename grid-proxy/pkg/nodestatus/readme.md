# About the nodestatus decision

Nodes periodically report their status to the chain:

- Normally up nodes report every 40 mins, have margin 2 cycles (80 mins) to report before they are marked down
- Farmerbot wake up the node once a day, if the node didn't report for 1 cycle (24 hours) it's marked down

Node status is determined based on:

- node.power state and target (up/down)
- updated_at timestamp if in the upInterval or standbyInterval or out of both intervals

Node status is determined as:

- up: if its updated_at in the last 80 mins and its power state/target is up (or null which means doesn't have farmerbot data).
- standby: if its updated_at in the last 24 hours and one or both of its power state/target is down
- down:
  - if its updated_at is older than 80 mins and its power state/target is up (or null)
  - if its updated_at is older than a full day

Some Definitions of used terms:

- nodeUpInterval: the duration in seconds that an UP node should report in.
- nodeStandbyInterval: the duration in seconds that a STANDBY node should report in.
- nilPower: node.power is null (is not powered by farmerbot)
- poweredOn: both node state and desired state are up
- poweredOff: both node state and desired state are down
- poweringOn: node state is down but farmerbot is trying to power it on
- poweringOff: node state is up but farmerbot is trying to power it off
