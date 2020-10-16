## test

every module has a test file named as \[moduleName\]_test.go, like <b>accounts_test.go</b> tests on account related operations

#### development

Go has a robust built-in testing library [testing](https://golang.org/pkg/testing/) 

#### usage

run a single test case

```
    $go test -run <TestName>
```

##### implementation with SetUp and TearDown  
setUp would register for org(s) and user(s); tearDown would hard remove all registered organizations

e.g.
```
func TestFoo(t *testing.T) {
	// set up and tear down
	registeredOrgs, registeredOrgUsers, orgids, err := testSetup(1, 2) // 1: number of orgs, 2: number of users per org
	assert.NilError(t, err)
	defer testTeardown(orgids)

	t.Run("example test", func(t *testing.T) {
		//start test
		org1 := registeredOrgs[0]
		usersInOrg1 := registeredOrgUsers[0]
		org1Admin := usersInOrg1[0] // the first user is admin
		org1User := usersInOrg1[1]
		assert.Check(t, org1User.OrgID == org1Admin.OrgID && org1User.OrgID == org1.OrgID)
	})

}
```


#### Troubleshooting

Failed to TearDown

----
tearDown hard remove registered organizations via superUser API  
- make sure config file(<b>dev.json</b>) is configured correctly and Internal Serive is enabled on server side  
- the failure of tearDown would raise some unexpected problems, remember to drop the db before restart