package test

import (
	"fmt"
	"github.com/overnest/strongdoc-go-sdk/test/testUtils"
	"log"
	"os"
	"testing"

	"github.com/overnest/strongdoc-go-sdk/api"
	"github.com/overnest/strongdoc-go-sdk/client"
	"gotest.tools/assert"
)

/**
tests for sharing
run all tests $go test -run TestShare
*/

const TestDocName = "TestDoc"
const TestSearchQuery = "Privacy"

func addSharableOrg(sdc client.StrongDocClient, user *testUtils.TestUser, org *testUtils.TestOrg) (succ bool, err error) {
	err = api.Login(sdc, user.UserID, user.Password, user.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	succ, err = api.AddSharableOrg(sdc, org.OrgID)

	return
}

func setMultiLevelShare(sdc client.StrongDocClient, user *testUtils.TestUser) (succ bool, err error) {
	// org1 admin set multiLevelShare enabled and add sharableOrg
	err = api.Login(sdc, user.UserID, user.Password, user.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	return api.SetMultiLevelSharing(sdc, true)
}

// share document with another user
func shareWithUser(sdc client.StrongDocClient, docID string, fromUser *testUtils.TestUser, toUserIDorEmail string) (isAccessible, isSharable bool, sharedReceivers, alreadyAccessibleUsers, unsharableUsers map[string]bool, err error) {
	err = api.Login(sdc, fromUser.UserID, fromUser.Password, fromUser.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	return api.BulkShareDocWithUsers(sdc, docID, []string{toUserIDorEmail})
}

// unshare document with another user
func unshareWithUser(sdc client.StrongDocClient, docID string, fromUser *testUtils.TestUser, UserIDorEmail string) (succ bool, alreadyNoAccess, allowed bool, err error) {
	err = api.Login(sdc, fromUser.UserID, fromUser.Password, fromUser.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	return api.UnshareWithUser(sdc, docID, UserIDorEmail)
}

// share document with another org
func shareWithOrg(sdc client.StrongDocClient, docID string, fromUser *testUtils.TestUser, toOrg *testUtils.TestOrg) (isAccessible, isSharable bool, sharedReceivers, alreadyAccessibleOrgs, unsharableOrgs map[string]bool, err error) {
	err = api.Login(sdc, fromUser.UserID, fromUser.Password, fromUser.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	return api.BulkShareDocWithOrgs(sdc, docID, []string{toOrg.OrgID})
}

// unshare document with another org
func unshareWithOrg(sdc client.StrongDocClient, docID string, fromUser *testUtils.TestUser, orgID string) (succ bool, alreadyNoAccess, allowed bool, err error) {
	err = api.Login(sdc, fromUser.UserID, fromUser.Password, fromUser.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	return api.UnshareWithOrg(sdc, docID, orgID)
}

func checkDocAccess(t *testing.T, sdc client.StrongDocClient, docID string, testUser *testUtils.TestUser) (bool, error) {
	err := api.Login(sdc, testUser.UserID, testUser.Password, testUser.OrgID)
	if err != nil {
		return false, err
	}
	defer api.Logout(sdc)
	docs, err := api.ListDocuments(sdc)
	if err != nil {
		return false, err
	}
	assert.NilError(t, err)
	for _, doc := range docs {
		if doc.DocID == docID {
			return true, nil
		}
	}
	return false, nil
}

func search(sdc client.StrongDocClient, user *testUtils.TestUser, query string) (hits []*api.DocumentResult, err error) {
	err = api.Login(sdc, user.UserID, user.Password, user.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	return api.Search(sdc, query)
}

func uploadDocument(sdc client.StrongDocClient, uploader *testUtils.TestUser, E2E bool) (uploadDocID string, err error) {
	err = api.Login(sdc, uploader.UserID, uploader.Password, uploader.OrgID)
	if err != nil {
		return
	}
	defer api.Logout(sdc)
	file, err := os.Open(TestDoc1)
	defer file.Close()
	if E2E {
		uploadDocID, err = api.E2EEUploadDocument(sdc, TestDocName, file)
		log.Println("upload client-encrypted document DocID = ", uploadDocID)

	} else {
		uploadDocID, err = api.UploadDocumentStream(sdc, TestDocName, file)
		log.Println("upload server-encrypted document DocID = ", uploadDocID)
	}
	return
}

func testShareWithoutMultiLevelShare(t *testing.T, sdc client.StrongDocClient, registeredOrgs []*testUtils.TestOrg, registeredOrgUsers [][]*testUtils.TestUser, E2E bool) {
	org1Admin := registeredOrgUsers[0][0]
	org1User1 := registeredOrgUsers[0][1]
	org1User2 := registeredOrgUsers[0][2]

	org2 := registeredOrgs[1]
	org2Admin := registeredOrgUsers[1][0]
	org2User := registeredOrgUsers[1][1]

	org3 := registeredOrgs[2]
	org3Admin := registeredOrgUsers[2][0]

	// org1User1(uploader) upload a file, only org1Admin and org1User1 can share the document
	docID, err := uploadDocument(sdc, org1User1, E2E)
	assert.NilError(t, err)

	// ----------------------------- share with user within same org -----------------------------
	// share
	_, _, _, _, _, err = shareWithUser(sdc, docID, org1Admin, org1User2.UserID)
	assert.NilError(t, err)
	access, err := checkDocAccess(t, sdc, docID, org1User2)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org1User2.Name, docID))

	// share again
	_, _, _, alreadySharedUsers, _, err := shareWithUser(sdc, docID, org1Admin, org1User2.Email)
	assert.Check(t, alreadySharedUsers[org1User2.Email], fmt.Sprintf("user %v already has access to the document %v", org1User2.Name, docID))

	// fail to unshare with uploader
	_, _, allowed, err := unshareWithUser(sdc, docID, org1Admin, org1User1.UserID)
	assert.Check(t, !allowed, fmt.Sprintf("user %v is not allowed to unshare %v", org1Admin.Name, org1User1.Name))

	// unshare
	succ, _, _, err := unshareWithUser(sdc, docID, org1Admin, org1User2.UserID)
	assert.Check(t, succ, fmt.Sprintf("user %v fail to unshare %v", org1Admin.Name, org1User2.Name))

	// unshare again
	_, alreadyNoAccess, _, err := unshareWithUser(sdc, docID, org1Admin, org1User2.UserID)
	assert.Check(t, alreadyNoAccess, fmt.Sprintf("user %v already have no access to doc %v", org1User2.Name, docID))

	access, err = checkDocAccess(t, sdc, docID, org1User2)
	assert.Check(t, !access, fmt.Sprintf("user %v should have no access to doc %v after unsharing", org1User2.Name, docID))

	// ----------------------------- share with user from external org -----------------------------
	// fail because org2 is unsharable
	_, _, _, _, unsharableUsers, err := shareWithUser(sdc, docID, org1Admin, org2User.UserID)
	assert.NilError(t, err)
	assert.Check(t, unsharableUsers[org2User.UserID], fmt.Sprintf("user %v should be unsharable", org2User.Name))

	succ, err = addSharableOrg(sdc, org1Admin, org2)
	assert.Check(t, succ && err == nil, fmt.Sprintf("failed to add sharableOrg %v", org2.OrgID))

	// share
	_, _, _, _, _, err = shareWithUser(sdc, docID, org1Admin, org2User.Email)
	assert.NilError(t, err)
	access, err = checkDocAccess(t, sdc, docID, org2User)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org2User.Name, docID))

	access, err = checkDocAccess(t, sdc, docID, org2Admin)
	assert.NilError(t, err)
	assert.Check(t, !access, fmt.Sprintf("user %v cannot have access to doc %v after sharing", org2User.Name, docID))

	// share again
	_, _, _, alreadySharedUsers, _, err = shareWithUser(sdc, docID, org1Admin, org2User.UserID)
	assert.NilError(t, err)
	assert.Check(t, alreadySharedUsers[org2User.UserID], fmt.Sprintf("user %v already has access to the document %v", org2User.Email, docID))

	// unshare
	succ, _, _, err = unshareWithUser(sdc, docID, org1Admin, org2User.UserID)
	assert.Check(t, succ, fmt.Sprintf("user %v fail to unshare %v", org1Admin.Name, org2User.Name))

	// unshare again
	_, alreadyNoAccess, _, err = unshareWithUser(sdc, docID, org1Admin, org2User.UserID)
	assert.Check(t, alreadyNoAccess, fmt.Sprintf("user %v already have no access to doc %v", org2User.Name, docID))

	access, err = checkDocAccess(t, sdc, docID, org2User)
	assert.Check(t, !access, fmt.Sprintf("user %v should have no access to doc %v after unsharing", org2User.Name, docID))

	// ----------------------------- share with another org -----------------------------
	// fail because org3 is unsharable
	_, _, _, alreadyAccessibleOrgs, unsharableOrgs, err := shareWithOrg(sdc, docID, org1Admin, org3)
	assert.Check(t, unsharableOrgs[org3.OrgID], fmt.Sprintf("org %v should  be unsharable", org3.OrgID))

	succ, err = addSharableOrg(sdc, org1Admin, org3)
	assert.NilError(t, err)
	assert.Check(t, succ, fmt.Sprintf("failed to add sharableOrg %v", org3.OrgID))

	// share
	_, _, _, _, _, err = shareWithOrg(sdc, docID, org1Admin, org3)
	assert.NilError(t, err)
	access, err = checkDocAccess(t, sdc, docID, org3Admin)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org3Admin.Name, docID))

	// share again
	_, _, _, alreadyAccessibleOrgs, _, err = shareWithOrg(sdc, docID, org1Admin, org3)
	assert.Check(t, alreadyAccessibleOrgs[org3.OrgID], fmt.Sprintf("org %v already has access to the document", org3.OrgID))

	// unshare
	succ, _, _, err = unshareWithOrg(sdc, docID, org1Admin, org3.OrgID)
	assert.Check(t, succ, fmt.Sprintf("user %v fail to unshare org %v", org1Admin.Name, org3.OrgID))

	// unshare again
	_, alreadyNoAccess, _, err = unshareWithOrg(sdc, docID, org1Admin, org3.OrgID)
	assert.Check(t, alreadyNoAccess, fmt.Sprintf("org %v already have no access to doc %v", org3.OrgID, docID))

	access, err = checkDocAccess(t, sdc, docID, org3Admin)
	assert.Check(t, !access, fmt.Sprintf("user %v should have no access to doc %v after unsharing", org3Admin.Name, docID))
}

func testShareWithMultiLevelShare(t *testing.T, sdc client.StrongDocClient, registeredOrgs []*testUtils.TestOrg, registeredOrgUsers [][]*testUtils.TestUser, E2E bool) {
	org1Admin := registeredOrgUsers[0][0]
	org1User1 := registeredOrgUsers[0][1]
	org1User2 := registeredOrgUsers[0][2]
	org1User3 := registeredOrgUsers[0][3]

	org2 := registeredOrgs[1]
	org2User1 := registeredOrgUsers[1][1]

	org3 := registeredOrgs[2]
	org3Admin := registeredOrgUsers[2][0]

	// org1 admin set multiLevelShare enabled and add sharableOrg
	succ, err := setMultiLevelShare(sdc, org1Admin)
	assert.NilError(t, err)
	assert.Check(t, succ, "fail to enable multiLevelSharing")

	succ, err = addSharableOrg(sdc, org1Admin, org2)
	assert.NilError(t, err)
	assert.Check(t, succ, fmt.Sprintf("failed to add sharableOrg %v", org2.OrgID))

	succ, err = addSharableOrg(sdc, org1Admin, org3)
	assert.NilError(t, err)
	assert.Check(t, succ, fmt.Sprintf("failed to add sharableOrg %v", org3.OrgID))

	// org1User(uploader) upload a doc
	docID, err := uploadDocument(sdc, org1User1, E2E)
	assert.NilError(t, err)

	// ----------------------------- share with user within same org -----------------------------
	// org1User1(uploader) share doc with org1User2
	_, _, _, _, _, err = shareWithUser(sdc, docID, org1User1, org1User2.UserID)
	assert.NilError(t, err)
	access, err := checkDocAccess(t, sdc, docID, org1User2)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org1User2.Name, docID))

	// org1User2 share doc with internal user
	_, _, _, _, _, err = shareWithUser(sdc, docID, org1User2, org1User3.UserID)
	assert.NilError(t, err)
	access, err = checkDocAccess(t, sdc, docID, org1User3)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org1User3.Name, docID))

	// fail to unshare with uploader
	_, _, allowed, err := unshareWithUser(sdc, docID, org1Admin, org1User1.UserID)
	assert.Check(t, !allowed, fmt.Sprintf("user %v is not allowed to unshare %v", org1Admin.Name, org1User1.Name))

	// unshare
	succ, _, _, err = unshareWithUser(sdc, docID, org1Admin, org1User3.UserID)
	assert.Check(t, succ, fmt.Sprintf("user %v fail to unshare %v", org1Admin.Name, org1User3.Name))

	// unshare again
	_, alreadyNoAccess, _, err := unshareWithUser(sdc, docID, org1Admin, org1User3.UserID)
	assert.Check(t, alreadyNoAccess, fmt.Sprintf("user %v already have no access to doc %v", org1User3.Name, docID))

	access, err = checkDocAccess(t, sdc, docID, org1User3)
	assert.Check(t, !access, fmt.Sprintf("user %v should have no access to doc %v after unsharing", org1User3.Name, docID))

	// ----------------------------- share with user from external org -----------------------------
	// org1User2 share doc with external user
	_, _, _, _, _, err = shareWithUser(sdc, docID, org1User2, org2User1.UserID)
	assert.NilError(t, err)
	access, err = checkDocAccess(t, sdc, docID, org2User1)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org2User1.Name, docID))

	// unshare
	succ, _, _, err = unshareWithUser(sdc, docID, org1Admin, org2User1.UserID)
	assert.Check(t, succ, fmt.Sprintf("user %v fail to unshare %v", org1Admin.Name, org2User1.Name))

	// unshare again
	_, alreadyNoAccess, _, err = unshareWithUser(sdc, docID, org1Admin, org2User1.UserID)
	assert.Check(t, alreadyNoAccess, fmt.Sprintf("user %v already have no access to doc %v", org2User1.Name, docID))

	access, err = checkDocAccess(t, sdc, docID, org2User1)
	assert.Check(t, !access, fmt.Sprintf("user %v should have no access to doc %v after unsharing", org2User1.Name, docID))

	// ----------------------------- share with another org -----------------------------
	// org1Admin share doc with another org
	_, _, _, _, _, err = shareWithOrg(sdc, docID, org1Admin, org3)
	assert.NilError(t, err)
	access, err = checkDocAccess(t, sdc, docID, org3Admin)
	assert.NilError(t, err)
	assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org3Admin.Name, docID))

	// unshare
	succ, _, _, err = unshareWithOrg(sdc, docID, org1Admin, org3.OrgID)
	assert.Check(t, succ, fmt.Sprintf("user %v fail to unshare org %v", org1Admin.Name, org3.OrgID))

	// unshare again
	_, alreadyNoAccess, _, err = unshareWithOrg(sdc, docID, org1Admin, org3.OrgID)
	assert.Check(t, alreadyNoAccess, fmt.Sprintf("org %v already have no access to doc %v", org3.OrgID, docID))

	access, err = checkDocAccess(t, sdc, docID, org3Admin)
	assert.Check(t, !access, fmt.Sprintf("user %v should have no access to doc %v after unsharing", org3Admin.Name, docID))
}

/**
set multiLevelShare to false
only uploader or admin of uploader's org can share the document
*/
func TestShare1(t *testing.T) {
	log.Println("test share server-encrypted document with multiLevelShare disabled")
	sdc, registeredOrgs, registeredOrgUsers := testUtils.PrevTest(t, 3, 3)
	testUtils.DoRegistration(t, sdc, registeredOrgs, registeredOrgUsers)
	t.Run("test share server-encrypted document with multiLevelShare disabled", func(t *testing.T) {
		testShareWithoutMultiLevelShare(t, sdc, registeredOrgs, registeredOrgUsers, false)
	})
}

func TestShare2(t *testing.T) {
	log.Println("test share client-encrypted document with multiLevelShare disabled")
	sdc, registeredOrgs, registeredOrgUsers := testUtils.PrevTest(t, 3, 3)
	testUtils.DoRegistration(t, sdc, registeredOrgs, registeredOrgUsers)
	t.Run("test share client-encrypted document with multiLevelShare disabled", func(t *testing.T) {
		testShareWithoutMultiLevelShare(t, sdc, registeredOrgs, registeredOrgUsers, true)
	})
}

/**
set multiLevelShare to true
anyone within uploader's organization who has access to the document
(including uploader, admin(s) of uploader's organization or someone else who has been given access)
*/
func TestShare3(t *testing.T) {
	log.Println("test share server-encrypted document with multiLevelShare enabled")
	sdc, registeredOrgs, registeredOrgUsers := testUtils.PrevTest(t, 3, 4)
	testUtils.DoRegistration(t, sdc, registeredOrgs, registeredOrgUsers)
	t.Run("test share server-encrypted document with multiLevelShare enabled", func(t *testing.T) {
		testShareWithMultiLevelShare(t, sdc, registeredOrgs, registeredOrgUsers, false)
	})
}

func TestShare4(t *testing.T) {
	log.Println("test share client-encrypted document with multiLevelShare enabled")
	sdc, registeredOrgs, registeredOrgUsers := testUtils.PrevTest(t, 3, 4)
	testUtils.DoRegistration(t, sdc, registeredOrgs, registeredOrgUsers)
	t.Run("test share client-encrypted document with multiLevelShare enabled", func(t *testing.T) {
		testShareWithMultiLevelShare(t, sdc, registeredOrgs, registeredOrgUsers, true)
	})
}

// test server-encrypted document search after sharing
func TestShareSearch(t *testing.T) {
	log.Println("test document search after sharing")
	sdc, registeredOrgs, registeredOrgUsers := testUtils.PrevTest(t, 3, 3)
	testUtils.DoRegistration(t, sdc, registeredOrgs, registeredOrgUsers)
	t.Run("test document search", func(t *testing.T) {
		org1Admin := registeredOrgUsers[0][0]
		org1User1 := registeredOrgUsers[0][1]
		org1User2 := registeredOrgUsers[0][2]

		org2 := registeredOrgs[1]
		org2Admin := registeredOrgUsers[1][0]
		org2User := registeredOrgUsers[1][1]

		org3 := registeredOrgs[2]
		org3Admin := registeredOrgUsers[2][0]
		org3User := registeredOrgUsers[2][1]

		// org1User1(uploader) upload a file
		docID, err := uploadDocument(sdc, org1User1, false)
		assert.NilError(t, err)

		hits, err := search(sdc, org1User1, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 1, fmt.Sprintf("uploader %v fail to search", org1User1.Name))

		hits, err = search(sdc, org1Admin, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 1, fmt.Sprintf("admin of uploader's org %v fail to search", org1Admin.Name))

		// ----------------------------- share with user within same org -----------------------------
		_, _, _, _, _, err = shareWithUser(sdc, docID, org1User1, org1User2.UserID)
		assert.NilError(t, err)
		access, err := checkDocAccess(t, sdc, docID, org1User2)
		assert.NilError(t, err)
		assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org1User2.Name, docID))

		hits, err = search(sdc, org1User2, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 1, fmt.Sprintf("shared user %v fail to search", org1User2.Name))

		_, _, _, err = unshareWithUser(sdc, docID, org1User1, org1User2.Email)
		assert.NilError(t, err)
		hits, err = search(sdc, org1User2, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 0, fmt.Sprintf("unshared user %v cannot search", org1User2.Name))

		// ----------------------------- share with user from external org -----------------------------
		succ, err := addSharableOrg(sdc, org1Admin, org2)
		assert.Check(t, succ && err == nil, fmt.Sprintf("failed to add sharableOrg %v", org2.OrgID))
		_, _, _, _, _, err = shareWithUser(sdc, docID, org1User1, org2User.Email)
		assert.NilError(t, err)
		access, err = checkDocAccess(t, sdc, docID, org2User)
		assert.NilError(t, err)
		assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org2User.Name, docID))

		hits, err = search(sdc, org2User, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 1, fmt.Sprintf("shared user %v fail to search", org2User.Name))

		hits, err = search(sdc, org2Admin, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 0, fmt.Sprintf("unshared user %v cannot search", org2Admin.Name))

		_, _, _, err = unshareWithUser(sdc, docID, org1User1, org2User.Email)
		assert.NilError(t, err)
		hits, err = search(sdc, org2User, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 0, fmt.Sprintf("unshared user %v cannot search", org2User.Name))

		// ----------------------------- share with another org -----------------------------
		succ, err = addSharableOrg(sdc, org1Admin, org3)
		assert.NilError(t, err)
		assert.Check(t, succ, fmt.Sprintf("failed to add sharableOrg %v", org3.OrgID))
		_, _, _, _, _, err = shareWithOrg(sdc, docID, org1Admin, org3)
		assert.NilError(t, err)
		access, err = checkDocAccess(t, sdc, docID, org3Admin)
		assert.NilError(t, err)
		assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org3Admin.Name, docID))

		hits, err = search(sdc, org3Admin, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 1, fmt.Sprintf("shared org admin %v fail to search", org3Admin.Name))

		hits, err = search(sdc, org3User, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 0, fmt.Sprintf("unshared user %v cannot search", org3User.Name))

		_, _, _, _, _, err = shareWithUser(sdc, docID, org1User1, org3User.Email)
		assert.NilError(t, err)
		access, err = checkDocAccess(t, sdc, docID, org3User)
		assert.NilError(t, err)
		assert.Check(t, access, fmt.Sprintf("user %v should have access to doc %v after sharing", org3User.Name, docID))

		hits, err = search(sdc, org3User, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 1, fmt.Sprintf("shared user %v fail to search", org3User.Name))

		_, _, _, err = unshareWithUser(sdc, docID, org1User1, org3User.Email)
		assert.NilError(t, err)
		hits, err = search(sdc, org3User, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 0, fmt.Sprintf("unshared user %v cannot search", org3User.Name))

		_, _, _, err = unshareWithOrg(sdc, docID, org1User1, org3.OrgID)
		assert.NilError(t, err)
		hits, err = search(sdc, org3Admin, TestSearchQuery)
		assert.NilError(t, err)
		assert.Check(t, len(hits) == 0, fmt.Sprintf("unshared user %v cannot search", org3User.Name))
	})
}
