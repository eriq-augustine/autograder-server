package main

import (
    "fmt"

    "github.com/alecthomas/kong"
    "github.com/rs/zerolog/log"

    "github.com/eriq-augustine/autograder/config"
    "github.com/eriq-augustine/autograder/db"
    "github.com/eriq-augustine/autograder/scoring"
)

var args struct {
    config.ConfigArgs
    Course string `help:"ID of the course." arg:""`
    Assignment string `help:"ID of the assignment." arg:""`
    DryRun bool `help:"Do not actually upload the grades, just state what you would do." default:"false"`
}

func main() {
    kong.Parse(&args,
        kong.Description("Perform a full assignment scoring (including late policy) and upload."),
    );

    err := config.HandleConfigArgs(args.ConfigArgs);
    if (err != nil) {
        log.Fatal().Err(err).Msg("Could not load config options.");
    }

    db.MustOpen();
    defer db.MustClose();

    assignment := db.MustGetAssignment(args.Course, args.Assignment);
    if (assignment.GetLMSID() == "") {
        log.Fatal().Msg("Assignment has no LMS ID.");
    }

    err = scoring.FullAssignmentScoringAndUpload(assignment, args.DryRun);
    if (err != nil) {
        log.Fatal().Err(err).Msg("Failed to score and upload assignment.");
    }


    fmt.Println("Assignment grades uploaded.");
}
