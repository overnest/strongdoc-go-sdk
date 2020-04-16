package api

// GetConfiguration returns a string representing the configurations
// set.
// func GetConfiguration() (config string, err error) {
// 	// authConn, err := client.ConnectToServerWithAuth(token)
// 	// if err != nil {
// 	// 	log.Fatalf("Can not obtain auth connection %s", err)
// 	// 	return
// 	// }
// 	// defer authConn.Close()

// 	// authClient := proto.NewStrongDocServiceClient(authConn)

// 	sdc, err := client.GetStrongDocClient()
// 	if err != nil {
// 		return
// 	}

// 	req := &proto.GetConfigurationReq{}
// 	res, err := sdc.GetConfiguration(context.Background(), req)
// 	if err != nil {
// 		return
// 	}
// 	config = res.Configuration
// 	return
// }
