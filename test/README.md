## test


#### development
We use Package [testing](https://golang.org/pkg/testing/) to write unit tests and the go test command runs tests

#### usage
func TestMain() defines some setup work before running test(s) , also clean up after down with all specified testing case(s).   

we implemented a flag parser, allowing user to choose:

with <b>-option=dev</b>, running setup before testing and tearDown after all work is done  
```
    go test -run <TestName> -option=dev
```
otherwise, directly call the test case itself
```
    go test -run <TestName>
```

#### Config
setUp registered for organization(s) and user(s), tearDown hard remove all organizations via superUser API.  
Please fill in config file(<b>dev.json</b>) or export environment variables as required before testing with option  

```
    $export SUPER_USER_ID=<super user id>
    $export SUPER_USER_PASSWORD=<super user password>
```

Also, make sure Internal Serive is enabled on server side

