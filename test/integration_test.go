package test

// const (
// 	AdminName         = "adminUserName"
// 	AdminPassword     = "adminUserPassword"
// 	AdminEmail        = "adminUser@somewhere.com"
// 	Organization      = "OrganizationOne"
// 	OrganizationAddr  = ""
// 	OrganizationEmail = "info@organizationone.com"
// 	Source            = "Test Active"
// 	SourceData        = ""
// 	TestDoc1          = "../testDocuments/CompanyIntro.txt"
// 	TestDoc2          = "../testDocuments/BedMounts.pdf"
// 	LargeDocFileName  = "/tmp/largeDoc.txt"
// 	TestText          = `The unanimous Declaration of the thirteen united States of America, When in the Course of human events, it becomes necessary for one people to dissolve the political bands which have connected them with another, and to assume among the powers of the earth, the separate and equal station to which the Laws of Nature and of Nature's God entitle them, a decent respect to the opinions of mankind requires that they should declare the causes which impel them to the separation. We hold these truths to be self-evident, that all men are created equal, that they are endowed by their Creator with certain unalienable Rights, that among these are Life, Liberty and the pursuit of Happiness.--That to secure these rights, Governments are instituted among Men, deriving their just powers from the consent of the governed, --That whenever any Form of Government becomes destructive of these ends, it is the Right of the People to alter or to abolish it, and to institute new Government, laying its foundation on such principles and organizing its powers in such form, as to them shall seem most likely to effect their Safety and Happiness. Prudence, indeed, will dictate that Governments long established should not be changed for light and transient causes; and accordingly all experience hath shewn, that mankind are more disposed to suffer, while evils are sufferable, than to right themselves by abolishing the forms to which they are accustomed. But when a long train of abuses and usurpations, pursuing invariably the same Object evinces a design to reduce them under absolute Despotism, it is their right, it is their duty, to throw off such Government, and to provide new Guards for their future security.--Such has been the patient sufferance of these Colonies; and such is now the necessity which constrains them to alter their former Systems of Government. The history of the present King of Great Britain is a history of repeated injuries and usurpations, all having in direct object the establishment of an absolute Tyranny over these States. To prove this, let Facts be submitted to a candid world. He has refused his Assent to Laws, the most wholesome and necessary for the public good. He has forbidden his Governors to pass Laws of immediate and pressing importance, unless suspended in their operation till his Assent should be obtained; and when so suspended, he has utterly neglected to attend to them.
// He has refused to pass other Laws for the accommodation of large districts of people, unless those people would relinquish the right of Representation in the Legislature, a right inestimable to them and formidable to tyrants only.
// He has called together legislative bodies at places unusual, uncomfortable, and distant from the depository of their public Records, for the sole purpose of fatiguing them into compliance with his measures.
// He has dissolved Representative Houses repeatedly, for opposing with manly firmness his invasions on the rights of the people.
// He has refused for a long time, after such dissolutions, to cause others to be elected; whereby the Legislative powers, incapable of Annihilation, have returned to the People at large for their exercise; the State remaining in the mean time exposed to all the dangers of invasion from without, and convulsions within.
// He has endeavoured to prevent the population of these States; for that purpose obstructing the Laws for Naturalization of Foreigners; refusing to pass others to encourage their migrations hither, and raising the conditions of new Appropriations of Lands.
// He has obstructed the Administration of Justice, by refusing his Assent to Laws for establishing Judiciary powers.
// He has made Judges dependent on his Will alone, for the tenure of their offices, and the amount and payment of their salaries.
// He has erected a multitude of New Offices, and sent hither swarms of Officers to harrass our people, and eat out their substance.
// He has kept among us, in times of peace, Standing Armies without the Consent of our legislatures.
// He has affected to render the Military independent of and superior to the Civil power.
// He has combined with others to subject us to a jurisdiction foreign to our constitution, and unacknowledged by our laws; giving his Assent to their Acts of pretended Legislation:
// For Quartering large bodies of armed troops among us:
// For protecting them, by a mock Trial, from punishment for any Murders which they should commit on the Inhabitants of these States:
// For cutting off our Trade with all parts of the world:
// For imposing Taxes on us without our Consent:
// For depriving us in many cases, of the benefits of Trial by Jury:
// For transporting us beyond Seas to be tried for pretended offences
// For abolishing the free System of English Laws in a neighbouring Province, establishing therein an Arbitrary government, and enlarging its Boundaries so as to render it at once an example and fit instrument for introducing the same absolute rule into these Colonies:
// For taking away our Charters, abolishing our most valuable Laws, and altering fundamentally the Forms of our Governments:
// For suspending our own Legislatures, and declaring themselves invested with power to legislate for us in all cases whatsoever.
// He has abdicated Government here, by declaring us out of his Protection and waging War against us.
// He has plundered our seas, ravaged our Coasts, burnt our towns, and destroyed the lives of our people.
// He is at this time transporting large Armies of foreign Mercenaries to compleat the works of death, desolation and tyranny, already begun with circumstances of Cruelty & perfidy scarcely paralleled in the most barbarous ages, and totally unworthy the Head of a civilized nation.
// He has constrained our fellow Citizens taken Captive on the high Seas to bear Arms against their Country, to become the executioners of their friends and Brethren, or to fall themselves by their Hands.
// He has excited domestic insurrections amongst us, and has endeavoured to bring on the inhabitants of our frontiers, the merciless Indian Savages, whose known rule of warfare, is an undistinguished destruction of all ages, sexes and conditions.
// In every stage of these Oppressions We have Petitioned for Redress in the most humble terms: Our repeated Petitions have been answered only by repeated injury. A Prince whose character is thus marked by every act which may define a Tyrant, is unfit to be the ruler of a free people.
// Nor have We been wanting in attentions to our Brittish brethren. We have warned them from time to time of attempts by their legislature to extend an unwarrantable jurisdiction over us. We have reminded them of the circumstances of our emigration and settlement here. We have appealed to their native justice and magnanimity, and we have conjured them by the ties of our common kindred to disavow these usurpations, which, would inevitably interrupt our connections and correspondence. They too have been deaf to the voice of justice and of consanguinity. We must, therefore, acquiesce in the necessity, which denounces our Separation, and hold them, as we hold the rest of mankind, Enemies in War, in Peace Friends.
// We, therefore, the Representatives of the united States of America, in General Congress, Assembled, appealing to the Supreme Judge of the world for the rectitude of our intentions, do, in the Name, and by Authority of the good People of these Colonies, solemnly publish and declare, That these United Colonies are, and of Right ought to be Free and Independent States; that they are Absolved from all Allegiance to the British Crown, and that all political connection between them and the State of Great Britain, is and ought to be totally dissolved; and that as Free and Independent States, they have full Power to levy War, conclude Peace, contract Alliances, establish Commerce, and to do all other Acts and Things which Independent States may of right do. And for the support of this Declaration, with a firm reliance on the protection of divine Providence, we mutually pledge to each other our Lives, our Fortunes and our sacred Honor.`
// )

// var sdc client.StrongDocClient

// func testAccounts(t *testing.T) {
// 	users, err := api.ListUsers(sdc)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, users)

// 	account, err := api.GetAccountInfo(sdc)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, account)

// 	fmt.Println("Account Info: ", account.OrgID, account.OrgAddress,
// 		account.Subscription.Status, account.Subscription.Type,
// 		account.MultiLevelShare)
// 	for _, org := range account.SharableOrgs {
// 		fmt.Println("    Sharable Org: ", org)
// 	}
// 	for _, pay := range account.Payments {
// 		fmt.Println("    Account Payments: ", pay.Status, pay.Amount,
// 			pay.BilledAt, pay.PeriodEnd, pay.PeriodStart)
// 	}

// 	user, err := api.GetUserInfo(sdc)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, user)
// }

// func testUploadDownload(t *testing.T, fileName string) (actions int) {
// 	actions = 0

// 	txtBytes, err := ioutil.ReadFile(fileName)
// 	assert.NoError(t, err)

// 	var uploads int = 6
// 	var uploadDocID string
// 	for i := 0; i < uploads; i++ {
// 		uploadDocID, err = api.UploadDocument(sdc, path.Base(fileName), txtBytes)
// 		assert.NoError(t, err)
// 		actions++
// 	}

// 	downBytes, err := api.DownloadDocument(sdc, uploadDocID)
// 	assert.NoError(t, err)
// 	assert.Equal(t, txtBytes, downBytes)
// 	actions++

// 	docs, err := api.ListDocuments(sdc)
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(docs), uploads)

// 	hits, err := api.Search(sdc, "security")
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(hits), uploads)

// 	err = api.RemoveDocument(sdc, uploadDocID)
// 	assert.NoError(t, err)
// 	actions++

// 	docs, err = api.ListDocuments(sdc)
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(docs), uploads-1)

// 	downBytes, err = api.DownloadDocument(sdc, uploadDocID)
// 	assert.Error(t, err)

// 	hits, err = api.Search(sdc, "security")
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(hits), uploads-1)

// 	file, err := os.Open(TestDoc1)
// 	assert.NoError(t, err)
// 	defer file.Close()

// 	uploadDocID, err = api.UploadDocumentStream(sdc, path.Base(fileName), file)
// 	assert.NoError(t, err)
// 	actions++

// 	stream, err := api.DownloadDocumentStream(sdc, uploadDocID)
// 	assert.NoError(t, err)
// 	downBytes, err = ioutil.ReadAll(stream)
// 	assert.NoError(t, err)
// 	assert.Equal(t, txtBytes, downBytes)
// 	actions++

// 	docs, err = api.ListDocuments(sdc)
// 	assert.NoError(t, err)
// 	for _, doc := range docs {
// 		err = api.RemoveDocument(sdc, doc.DocID)
// 		assert.NoError(t, err)
// 		actions++
// 	}

// 	return
// }

// func testEncryptDecrypt(t *testing.T, fileName string) (actions int) {
// 	actions = 0

// 	txtBytes, err := ioutil.ReadFile(fileName)
// 	assert.NoError(t, err)

// 	encryptDocID, ciphertext, err := api.EncryptDocument(sdc, path.Base(fileName), txtBytes)
// 	assert.NoError(t, err)

// 	plaintext, err := api.DecryptDocument(sdc, encryptDocID, ciphertext)
// 	assert.NoError(t, err)
// 	assert.Equal(t, txtBytes, plaintext)

// 	docs, err := api.ListDocuments(sdc)
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(docs), 1)

// 	hits, err := api.Search(sdc, "security")
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(hits), 1)

// 	err = api.RemoveDocument(sdc, encryptDocID)
// 	assert.NoError(t, err)
// 	actions++

// 	docs, err = api.ListDocuments(sdc)
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(docs), 0)

// 	plaintext, err = api.DecryptDocument(sdc, encryptDocID, ciphertext)
// 	assert.Error(t, err)

// 	hits, err = api.Search(sdc, "security")
// 	assert.NoError(t, err)
// 	assert.Equal(t, len(hits), 0)

// 	file, err := os.Open(TestDoc1)
// 	assert.NoError(t, err)
// 	defer file.Close()

// 	cipherStream, encryptDocID, err := api.EncryptDocumentStream(sdc, path.Base(fileName), file)
// 	assert.NoError(t, err)

// 	plainStream, err := api.DecryptDocumentStream(sdc, encryptDocID, cipherStream)
// 	assert.NoError(t, err)

// 	plaintext, err = ioutil.ReadAll(plainStream)
// 	assert.NoError(t, err)
// 	assert.Equal(t, txtBytes, plaintext)

// 	docs, err = api.ListDocuments(sdc)
// 	assert.NoError(t, err)
// 	for _, doc := range docs {
// 		err = api.RemoveDocument(sdc, doc.DocID)
// 		assert.NoError(t, err)
// 		actions++
// 	}

// 	return
// }

// func testBilling(t *testing.T) {
// 	bill, err := api.GetBillingDetails(sdc)
// 	assert.NoError(t, err)

// 	fmt.Println("Bill:", bill)
// 	fmt.Println("Bill-Documents:", bill.Documents)
// 	fmt.Println("Bill-Search:", bill.Search)
// 	fmt.Println("Bill-Traffic:", bill.Traffic)

// 	freqs, err := api.GetBillingFrequencyList(sdc)
// 	assert.NoError(t, err)
// 	for i, freq := range freqs {
// 		fmt.Printf("Get Frequency[%v]: %v\n", i, freq)
// 	}

// 	// Wait a bit
// 	time.Sleep(time.Second * 2)

// 	freq, err := api.SetNextBillingFrequency(sdc, proto.TimeInterval_MONTHLY, time.Now().AddDate(0, 2, 0))
// 	assert.NoError(t, err)
// 	fmt.Printf("Set Frequency: %v\n", freq)

// 	freqs, err = api.GetBillingFrequencyList(sdc)
// 	assert.NoError(t, err)
// 	for i, freq := range freqs {
// 		fmt.Printf("Get Frequency[%v]: %v\n", i, freq)
// 	}

// 	traffic, err := api.GetLargeTraffic(sdc, time.Now())
// 	assert.NoError(t, err)
// 	fmt.Println("Large Traffic:", traffic)
// }

// func testDocActionHistory(t *testing.T, actions int) {
// 	time.Sleep(time.Second * 1)
// 	docActionList, totalResult, offset, err := api.ListDocActionHistory(sdc)
// 	assert.NoError(t, err)
// 	assert.Equal(t, actions, len(docActionList))
// 	assert.Equal(t, int32(actions), totalResult)
// 	assert.Equal(t, int32(0), offset)
// }

// // todo: fix after e2ee change
// /*
// func TestIntegrationSmall(t *testing.T) {
// 	var err error

// 	sdc, err = client.InitStrongDocClient(client.LOCAL, false)
// 	assert.NoError(t, err)

// 	orgID, _, err := api.RegisterOrganization(sdc, Organization, OrganizationAddr,
// 		OrganizationEmail, AdminName, AdminPassword, AdminEmail, Source, SourceData)
// 	assert.NoError(t, err)
// 	assert.Equal(t, orgID, Organization)

// 	token, err := api.Login(sdc, AdminEmail, AdminPassword, Organization)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, token)

// 	defer func() {
// 		success, err := api.RemoveOrganization(sdc, true)
// 		assert.NoError(t, err)
// 		assert.True(t, success)
// 	}()

// 	var actions int = 0

// 	testAccounts(t)
// 	actions += testUploadDownload(t, TestDoc1)
// 	testDocActionHistory(t, actions)
// 	actions += testEncryptDecrypt(t, TestDoc1)
// 	testDocActionHistory(t, actions)
// 	testBilling(t)

// 	_, err = api.Logout(sdc)
// 	assert.NoError(t, err)

// 	_, err = api.ListDocuments(sdc)
// 	assert.Error(t, err)

// 	// Need to wait at least 1 second before logging back in
// 	time.Sleep(time.Second * 2)

// 	// Log back in so organization can be removed
// 	token, err = api.Login(sdc, AdminEmail, AdminPassword, Organization)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, token)
// } */

// func uploadLargeDoc(t *testing.T, fileName string, size int64) string {
// 	file, err := os.Create(fileName)
// 	assert.NoError(t, err)
// 	defer file.Close()

// 	for size > 0 {
// 		n, err := file.Write(([]byte)(TestText))
// 		assert.NoError(t, err)
// 		size -= (int64)(n)
// 	}

// 	file.Close()
// 	file, err = os.Open(fileName)
// 	assert.NoError(t, err)

// 	uploadDocID, err := api.UploadDocumentStream(sdc, path.Base(fileName), file)
// 	assert.NoError(t, err)

// 	fmt.Println("Upload Doc ID:", uploadDocID)
// 	return uploadDocID
// }

// func downloadLargDoc(t *testing.T, fileName, docID string) {
// 	file, err := os.Create(fileName)
// 	assert.NoError(t, err)
// 	defer file.Close()

// 	downloadStream, err := api.DownloadDocumentStream(sdc, docID)
// 	n, err := io.Copy(file, downloadStream)
// 	assert.NoError(t, err)

// 	fmt.Println("Download Doc ID:", docID, n)
// }

// // todo: fix after e2ee change
// /*
// func TestLargeFileUploadDownload(t *testing.T) {
// 	if testing.Short() {
// 		t.SkipNow()
// 	}

// 	fileSize := int64(100 * 1024 * 1024)

// 	var err error
// 	sdc, err = client.InitStrongDocClient(client.LOCAL, false)
// 	assert.NoError(t, err)

// 	orgID, _, err := api.RegisterOrganization(sdc, Organization, OrganizationAddr,
// 		OrganizationEmail, AdminName, AdminPassword, AdminEmail, Source, SourceData)
// 	assert.NoError(t, err)
// 	assert.Equal(t, orgID, Organization)

// 	token, err := api.Login(sdc, AdminEmail, AdminPassword, Organization)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, token)

// 	// Upload big file
// 	defer os.Remove(LargeDocFileName)
// 	uploadDocID := uploadLargeDoc(t, LargeDocFileName, fileSize)

// 	// Download big file
// 	downloadFileName := LargeDocFileName + "_down"
// 	defer os.Remove(downloadFileName)
// 	downloadLargDoc(t, downloadFileName, uploadDocID)

// 	equal, err := equalfile.New(nil, equalfile.Options{}).CompareFile(LargeDocFileName, downloadFileName)
// 	assert.Equal(t, equal, true)
// } */
