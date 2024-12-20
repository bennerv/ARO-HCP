package generate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/cobra"

	options "github.com/Azure/ARO-HCP/tooling/templatize/cmd"
	"github.com/Azure/ARO-HCP/tooling/templatize/pkg/ev2"
)

func DefaultGenerationOptions() *RawGenerationOptions {
	return &RawGenerationOptions{
		RolloutOptions: options.DefaultRolloutOptions(),
	}
}

func BindGenerationOptions(opts *RawGenerationOptions, cmd *cobra.Command) error {
	err := options.BindRolloutOptions(opts.RolloutOptions, cmd)
	if err != nil {
		return fmt.Errorf("failed to bind raw options: %w", err)
	}
	cmd.Flags().StringVar(&opts.Input, "input", opts.Input, "input file path")
	cmd.Flags().StringVar(&opts.Output, "output", opts.Output, "output file path")
	cmd.Flags().BoolVar(&opts.EV2Placeholders, "ev2-placeholders", opts.EV2Placeholders, "generate EV2 placeholders")

	for _, flag := range []string{"config-file", "input", "output"} {
		if err := cmd.MarkFlagFilename(flag); err != nil {
			return fmt.Errorf("failed to mark flag %q as a file: %w", flag, err)
		}
	}
	return nil
}

// RawGenerationOptions holds input values.
type RawGenerationOptions struct {
	RolloutOptions  *options.RawRolloutOptions
	Input           string
	Output          string
	EV2Placeholders bool
}

// validatedGenerationOptions is a private wrapper that enforces a call of Validate() before Complete() can be invoked.
type validatedGenerationOptions struct {
	*RawGenerationOptions
	*options.ValidatedRolloutOptions
}

type ValidatedGenerationOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*validatedGenerationOptions
}

// completedGenerationOptions is a private wrapper that enforces a call of Complete() before config generation can be invoked.
type completedGenerationOptions struct {
	*options.RolloutOptions
	InputFS    fs.FS
	InputFile  string
	OutputFile io.Writer
}

type GenerationOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedGenerationOptions
}

func (o *RawGenerationOptions) Validate() (*ValidatedGenerationOptions, error) {
	validatedRolloutOptions, err := o.RolloutOptions.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed for raw options: %w", err)
	}

	if _, err := os.Stat(o.Input); os.IsNotExist(err) {
		return nil, fmt.Errorf("input file %s does not exist", o.Input)
	}

	return &ValidatedGenerationOptions{
		validatedGenerationOptions: &validatedGenerationOptions{
			RawGenerationOptions:    o,
			ValidatedRolloutOptions: validatedRolloutOptions,
		},
	}, nil
}

func (o *ValidatedGenerationOptions) Complete() (*GenerationOptions, error) {
	completed, err := o.ValidatedRolloutOptions.Complete()
	if err != nil {
		return nil, err
	}

	if o.EV2Placeholders {
		_, vars := ev2.EV2Mapping(completed.Config, []string{})
		completed.Config = vars
	}

	inputFile := filepath.Base(o.Input)

	if err := os.MkdirAll(filepath.Dir(o.Output), os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", o.Output, err)
	}

	outputFile, err := os.Create(o.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", o.Input, err)
	}

	return &GenerationOptions{
		completedGenerationOptions: &completedGenerationOptions{
			RolloutOptions: completed,
			InputFS:        os.DirFS(filepath.Dir(o.Input)),
			InputFile:      inputFile,
			OutputFile:     outputFile,
		},
	}, nil
}

func (opts *GenerationOptions) ExecuteTemplate() error {
	tmpl := template.New(opts.InputFile).Funcs(sprig.FuncMap())
	content, err := fs.ReadFile(opts.InputFS, opts.InputFile)
	if err != nil {
		return err
	}

	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return err
	}

	return tmpl.Option("missingkey=error").ExecuteTemplate(opts.OutputFile, opts.InputFile, opts.RolloutOptions.Config)
}
