package model

import (
    "fmt"

    "github.com/eriq-augustine/autograder/lms"
    "github.com/eriq-augustine/autograder/usr"
    "github.com/eriq-augustine/autograder/util"
)

// Sync users with the provided LMS.
func (this *Course) SyncLMSUsers(dryRun bool, sendEmails bool) (*usr.UserSyncResult, error) {
    if (this.LMSAdapter == nil) {
        return nil, nil;
    }

    lmsUsersSlice, err := this.LMSAdapter.FetchUsers();
    if (err != nil) {
        return nil, fmt.Errorf("Failed to fetch LMS users: '%w'.", err);
    }

    lmsUsers := make(map[string]*lms.User, len(lmsUsersSlice));
    for _, lmsUser := range lmsUsersSlice {
        lmsUsers[lmsUser.Email] = lmsUser;
    }

    return this.syncLMSUsers(dryRun, sendEmails, lmsUsers, nil);
}

func (this *Course) SyncLMSUser(email string, dryRun bool, sendEmails bool) (*usr.UserSyncResult, error) {
    if (this.LMSAdapter == nil) {
        return nil, nil;
    }

    lmsUser, err := this.LMSAdapter.FetchUser(email);
    if (err != nil) {
        return nil, err;
    }

    lmsUsers := map[string]*lms.User{
        lmsUser.Email: lmsUser,
    };

    return this.syncLMSUsers(dryRun, sendEmails, lmsUsers, []string{email});
}

// Sync users.
// If |syncEmails| is not empty, then only emails in it will be checked/resolved.
// Otherwise, all emails from local and LMS users will be checked.
func (this *Course) syncLMSUsers(dryRun bool, sendEmails bool, lmsUsers map[string]*lms.User, syncEmails []string) (
        *usr.UserSyncResult, error) {
    localUsers, err := this.GetUsers();
    if (err != nil) {
        return nil, fmt.Errorf("Failed to fetch local users: '%w'.", err);
    }

    if (len(syncEmails) == 0) {
        syncEmails = getAllEmails(localUsers, lmsUsers);
    }

    syncResult := usr.NewUserSyncResult();

    for _, email := range syncEmails {
        resolveResult, err := this.resolveUserSync(localUsers, lmsUsers, email);
        if (err != nil) {
            return nil, err;
        }

        if (resolveResult != nil) {
            syncResult.AddResolveResult(resolveResult);
        }
    }

    if (dryRun) {
        return syncResult, nil;
    }

    err = this.SaveUsers(localUsers);
    if (err != nil) {
        return nil, fmt.Errorf("Failed to save users file: '%w'.", err);
    }

    if (sendEmails) {
        for _, newUser := range syncResult.Add {
            pass := syncResult.ClearTextPasswords[newUser.Email];
            usr.SendUserAddEmail(newUser, pass, true, false, dryRun, true);
        }
    }

    return syncResult, nil;
}

func mergeUsers(localUser *usr.User, lmsUser *lms.User, mergeAttributes bool) bool {
    changed := false;

    if (localUser.LMSID != lmsUser.ID) {
        localUser.LMSID = lmsUser.ID;
        changed = true;
    }

    if (!mergeAttributes) {
        return changed;
    }

    if (localUser.DisplayName != lmsUser.Name) {
        localUser.DisplayName = lmsUser.Name;
        changed = true;
    }

    if (localUser.Role != lmsUser.Role) {
        localUser.Role = lmsUser.Role;
        changed = true;
    }

    return changed;
}

// Resolve differences between a local user and LMS user (linked using the provided email).
// The passed in local user map will be modified to reflect any resolution.
// The taken action will depend on the options set in the course's LMS adapter.
func (this *Course) resolveUserSync(localUsers map[string]*usr.User, lmsUsers map[string]*lms.User, email string) (
        *usr.UserResolveResult, error) {
    localUser := localUsers[email];
    lmsUser := lmsUsers[email];

    if ((localUser == nil) && lmsUser == nil) {
        return nil, nil;
    }

    // Add.
    if (localUser == nil) {
        if (!this.LMSAdapter.SyncAddUsers) {
            return nil, nil;
        }

        pass, err := util.RandHex(usr.DEFAULT_PASSWORD_LEN);
        if (err != nil) {
            return nil, fmt.Errorf("Failed to generate a default password: '%w'.", err);
        }

        localUser = &usr.User{
            Email: email,
            DisplayName: lmsUser.Name,
            Role: lmsUser.Role,
            LMSID: lmsUser.ID,
        };

        hashPass := util.Sha256HexFromString(pass);
        err = localUser.SetPassword(hashPass);
        if (err != nil) {
            return nil, fmt.Errorf("Failed to set password: '%w'.", err);
        }

        localUsers[email] = localUser;

        return &usr.UserResolveResult{Add: localUser, ClearTextPassword: pass}, nil;
    }

    // Del.
    if (lmsUser == nil) {
        if (!this.LMSAdapter.SyncRemoveUsers) {
            return nil, nil;
        }

        delete(localUsers, email);
        return &usr.UserResolveResult{Del: localUser}, nil;
    }

    // Mod.
    userChanged := mergeUsers(localUser, lmsUser, this.LMSAdapter.SyncUserAttributes);
    if (userChanged) {
        return &usr.UserResolveResult{Mod: localUser}, nil;
    }

    return nil, nil;
}

func getAllEmails(localUsers map[string]*usr.User, lmsUsers map[string]*lms.User) []string {
    emails := make([]string, 0, max(len(localUsers), len(lmsUsers)));

    for email, _ := range localUsers {
        emails = append(emails, email);
    }

    for email, _ := range lmsUsers {
        _, ok := localUsers[email];
        if (ok) {
            // Already added.
            continue;
        }

        emails = append(emails, email);
    }

    return emails;
}
