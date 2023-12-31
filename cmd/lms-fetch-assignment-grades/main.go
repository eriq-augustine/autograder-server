package main

import (
    "fmt"
    "strings"

    "github.com/alecthomas/kong"
    "github.com/rs/zerolog/log"

    "github.com/eriq-augustine/autograder/config"
    "github.com/eriq-augustine/autograder/db"
    "github.com/eriq-augustine/autograder/lms"
    "github.com/eriq-augustine/autograder/util"
)

var args struct {
    config.ConfigArgs
    Course string `help:"ID of the course." arg:""`
    Assignment string `help:"ID of the assignment." arg:""`
}

func main() {
    kong.Parse(&args,
        kong.Description("Fetch the grades for a specific assignment from an LMS."),
    );

    err := config.HandleConfigArgs(args.ConfigArgs);
    if (err != nil) {
        log.Fatal().Err(err).Msg("Could not load config options.");
    }

    db.MustOpen();
    defer db.MustClose();

    assignment := db.MustGetAssignment(args.Course, args.Assignment);
    course := assignment.GetCourse();

    if (assignment.GetLMSID() == "") {
        log.Fatal().Str("assignment", assignment.FullID()).Msg("Assignment has no LMS ID.");
    }

    grades, err := lms.FetchAssignmentScores(course, assignment.GetLMSID());
    if (err != nil) {
        log.Fatal().Err(err).Msg("Could not fetch grades.");
    }

    fmt.Println("lms_user_id\tscore\ttime\tcomments");
    for _, grade := range grades {
        textComments := make([]string, 0, len(grade.Comments));
        for _, comment := range grade.Comments {
            textComments = append(textComments, comment.Text);
        }
        comments := strings.Join(textComments, ";");

        fmt.Printf("%s\t%s\t%s\t%s\n", grade.UserID, util.FloatToStr(grade.Score), grade.Time, comments);
    }
}
