# Filters & Fields Explanation

## Renting

Each node should be in one of three states

- shared
- rented
- rentable

The state is decided based on three factors

- node's farm dedicated flag
- node's node contract
- node's rent contract

Here is a truth table of all possibilities:

| node's farm dedicated flag | node's node contract | node's rent contract | Result             |
| -------------------------- | -------------------- | -------------------- | ------------------ |
| not dedicated              | doesn't have         | doesn't have         | Rentable Or Shared |
| not dedicated              | doesn't have         | have                 | Rented             |
| not dedicated              | have                 | doesn't have         | Shared             |
| not dedicated              | have                 | have                 | Rented             |
| is dedicated               | doesn't have         | doesn't have         | Rentable Only      |
| is dedicated               | doesn't have         | have                 | Rented             |
| ~~is dedicated~~           | ~~have~~             | ~~doesn't have~~     | Not valid          |
| is dedicated               | have                 | have                 | Rented             |

To sum up node is:

- Rentable: if it has no rent/node contract
- Rented: if it has a rent contract (no matter the node contract)
- Shared: if it doesn't have a rent contract and farm is not dedicated.

You can find the three states as field or filter for the nodes endpoint. there is also some extra useful filters/fields:

- dedicated_farm: shows if the node's farm is dedicated
- dedicated: shows rentable/rented nodes
- renter: the twin id of the renter if rented
- available_for: shows all shared nodes + nodes rented by a twin id
