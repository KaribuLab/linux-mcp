## ADDED Requirements

### Requirement: Shared recursive walk applies path policy per node with a hard node budget
The system MUST provide a shared in-process walk helper, usable by `find`, `grep`, and future tree-walking tools, that: (1) applies the same path denylist to every node visited during the walk, not only to the root path; (2) never follows symbolic links while descending the tree; and (3) enforces a hard budget on the total number of nodes visited (default 50,000), independent of how many results the caller ultimately requests or receives. When the node budget is reached, the walk MUST stop and the caller MUST be able to signal truncation to the client.

#### Scenario: Denied node skipped mid-walk
- **WHEN** the shared walk reaches a filesystem node that matches the path denylist, while the walk's root path is itself allowed
- **THEN** the walk MUST exclude that node from results and MUST continue visiting remaining nodes

#### Scenario: Walk never descends into a symlink target
- **WHEN** the shared walk encounters a symbolic link during traversal
- **THEN** the walk MUST NOT follow the link to descend into its target

#### Scenario: Walk stops at the node budget regardless of match count
- **WHEN** the shared walk visits a number of nodes equal to the configured node budget, whether or not any caller-defined criteria have matched
- **THEN** the walk MUST stop visiting further nodes and MUST report that the traversal was truncated by the node budget
