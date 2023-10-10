package main

import (
    "fmt"

    "github.com/alecthomas/kong"
    "github.com/rs/zerolog/log"

    "github.com/eriq-augustine/autograder/config"
    "github.com/eriq-augustine/autograder/model"
    "github.com/eriq-augustine/autograder/task"
    "github.com/eriq-augustine/autograder/util"
)

var args struct {
    config.ConfigArgs
    AssignmentPath string `help:"Path to assignment JSON file." arg:"" type:"existingfile"`
    HTML bool `help:"Output report as html." default:"false"`
}

func main() {
    kong.Parse(&args,
        kong.Description("Fetch the grades for a specific assignment from canvas."),
    );

    err := config.HandleConfigArgs(args.ConfigArgs);
    if (err != nil) {
        log.Fatal().Err(err).Msg("Could not load config options.");
    }

    assignment := model.MustLoadAssignmentConfig(args.AssignmentPath);

    report, err := task.GetScoringReport(assignment);
    if (err != nil) {
        log.Fatal().Err(err).Str("assignment", assignment.ID).Msg("Failed to get scoring report.");
    }

    if (args.HTML) {
        html, err := report.ToHTML();
        if (err != nil) {
            log.Fatal().Err(err).Str("assignment", assignment.ID).Msg("Failed to generate HTML scoring report.");
        }

        fmt.Println(html);
    } else {
        fmt.Println(util.MustToJSONIndent(report));
    }
}
