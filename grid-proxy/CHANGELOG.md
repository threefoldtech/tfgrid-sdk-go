# Changelog

Check `/version` on any instance to know the version.

## Projects

### 3.13

---

- include releases from v0.13.5 to v0.14.11

## Releases

### v0.15.2

---

- `fix` ipv6 indexer (default interval, rmb call result)

### v0.15.1

---

- `feat` add has_ipv6 indexer
- `feat` allow filtering with array of contract states
- `feat` filter nodes with num_gpu
- `feat` sorting nodes with free_cru

### v0.15.0

---

- `feat` refactor the indexer code using generics
- `feat` add network speed and dmi indexers
- `feat` add last-deployment-timestamp on nodes statistics call response

### v0.14.13

---

- `feat` add farm name/id to contract response
- `feat` optimize queries on stats endpoint

### v0.14.11

---

- `fix` fix the policy id 0 farms to be the default 1

### v0.14.8

---

- `feat` add rentable/rented fields in the node response.

### v0.14.7

---

- `fix` add trigger on node_gpu to update the node_gpu_count on cache.
- `fix` fix ordering by node status.

### v0.14.5

---

- `feat` add querying and sorting by node price

### v0.14.4

---

- `feat` add validation on query params for filter and limits.
- `fix` fix the check condition for inc/dec ips on cache.

### v0.14.2

---

- `fix` invalidate old gpus for each newly indexed node.

### v0.14.1

---

- `feat` add excluded filter for node endpoint

### v0.13.21

---

- `fix` fix node region filter to use region instead of subregion

### v0.13.18

---

- `patch` cherry-picks changes from the previous release v0.13.17 to continue v0.13.12

### v0.13.17

---

- `fix` fix the health indexer node querying to flip pages

### v0.13.12

---

- `patch` cherry-picks changes from the previous release v0.13.11 to continue v0.13.9

### v0.13.11

---

- `fix` fix node status filter for the broken `time.Now()`

### v0.13.9

---

- `patch` a custom release for the recent changes above 3.12 to reach mainnet without the upgraded runtime

### v0.13.7

---

- `fix` revalidate the health reports

### v0.13.6

---

- `fix` fix the node health indexer missing unique constrain

### v0.13.5

---

- `feat` introduce the health indexer
- `feat` add farmName to node response

### v0.13.4

---

- `feat` add node sorting by status

### v0.13.3

---

- `fix` fix the status filter broken by the null power object

### v0.13.0

---

- `feat` optimize database queries by denormalized tables
