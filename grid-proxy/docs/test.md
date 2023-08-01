# Gridproxy Tests

- To test gridproxy functionality, some integration tests are done. To carry these tests, the following needs to happen first:
  - a fake postgres db is run on a docker container to mimic the functionality of the real db that gridproxy connects to.
  - for the db to work properly, it has to have the same structure as the real db. this structure is defined in [schema.sql](../tools/db/schema.sql)
  - the [generate.go](../tools/db/generate.go) is run to apply the mentioned structure, and fill the db with pseudo-random data, i.e. if a field could only have certain values, only these values are considered while generating them.
- After preparing the database, tests can now run.
- The idea behind the tests is, if we make our queries with using two different approaches and if they match, the test passes, if not, the test fails.
- To do this, we need to have two clients to fullfil our queries. these two clients are the ProxyClient, and the MockClient.
- ProxyClient:
  - this is the client that we are trying to test. users who use gridproxy from Go programs use this client.
  - it simply makes the proper http call to the gridproxy server and returns back the response.
- MockClient:
  - this is a mock client that is built only for testing purposes.
  - it exposes the same api as the ProxyClient.
  - simply, this client loads all data from the database, and stores them in memory.
  - then, with the provided input, the MockClient decides which data should be returned.
  - the logic that the MockClient uses should reflect the user's needs, this is what you `want` the result to be, the ProxyClient is what you `got`.
- Since the only input a user provides is some kind of filter (`NodeFilter`, `FarmFilter`, `TwinFilter`, ...), and a `Limit`, then we need to validate the incoming result with different values of filters and limits.
- To test against random filter values, the tests provide a random value function for each field in a filter, like [here](../tests/queries/contract_test.go#L34)
- All filter fields must have random value functions referenced by the field name, otherwise tests will not pass.
- On the other hand, the MockClient should have a validator for nodes, farms, etc.. against filters, like [here](../tests/queries/mock_client/contracts.go#L11)

## Testing new changes

When adding new changes to the gridproxy, make sure to follow the below steps to properly test these changes:

- Check if there are any schema changes, if yes:
  - reflect these changes in [schema.sql](../tools/db/schema.sql)
  - reflect these changes in [generate.go](../tools//db/generate.go) to make sure generated data conform with the latest schema
- Update the MockClient's types and logic to reflect your changes, this is what you want the ProxyClient to do in the end.
