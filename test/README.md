## test


#### development
We use Package [testing](https://golang.org/pkg/testing/) to write unit tests and the go test command runs tests

#### usage
func TestMain() defines some setup work before running test(s) , also clean up after down with all specified testing case(s).   

we implemented a flag parser, allowing user to choose:

with <b>-dev</b>, running setup before testing and tearDown after all work is done  
```
    go test -run <TestName> -dev
```
without <b>-dev</b>, directly call the test case itself
```
    go test -run <TestName>
```

#### SetUp and TearDown
setUp registered for organization(s) and user(s), tearDown hard remove all organizations via superUser API.  
Please export environment variables as required before testing with <b>dev</b> option or you can modify the functions <b>getSuperUserId()</b> and <b>getSuperUserPwd()</b> locally.

```
    $export SUPER_USER_ID=<super user id>
    $export SUPER_USER_PASSWORD=<super user password>
```

