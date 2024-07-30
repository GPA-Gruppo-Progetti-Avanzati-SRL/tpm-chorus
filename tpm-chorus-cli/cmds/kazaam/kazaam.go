package kazaam

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/transform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/tpm-chorus-cli/cmds"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/fs"
	"os"
)

var (
	inputFile  string
	ruleFile   string
	outputFile string
)

const semLogContextCmd = "kazaam-cmd::"

// kazaamCmd
// Sample params: -i orchestration/transform/case-001-input.json -r orchestration/transform/case-001-rule.yml -o kazaam.out.json
var kazaamCmd = &cobra.Command{
	Use:   "kazaam",
	Short: "test of kazaam rules",
	Long:  `The command allows the application of kazaam rules to json data in order to test that the rule does what is expected to do.`,
	Run: func(cmd *cobra.Command, args []string) {

		const semLogContext = semLogContextCmd + "run"

		var err error
		err = validateArgs()
		if err != nil {
			return
		}

		err = transform.InitializeKazaamRegistry()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		inputData, err := os.ReadFile(inputFile)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		tid, err := registerRule(ruleFile)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		outData, err := transform.GetRegistry().Transform(tid, inputData)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return
		}

		if outputFile == "" {
			fmt.Println("############ kazaam output: ")
			fmt.Println(string(outData))
		} else {
			err = os.WriteFile(outputFile, outData, fs.ModePerm)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return
			}

			fmt.Printf("############ kazaam output: written to %s\n", outputFile)
		}
	},
}

func registerRule(ruleFile string) (string, error) {

	const semLogContext = semLogContextCmd + "register-rule"

	ruleData, err := os.ReadFile(ruleFile)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return "", err
	}

	trsf := transform.Config{}
	err = yaml.Unmarshal(ruleData, &trsf)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return "", err
	}

	err = transform.GetRegistry().Add(trsf)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return "", err
	}

	return trsf.Id, nil
}

func validateArgs() error {
	const semLogContext = semLogContextCmd + "validate-args"
	if inputFile == "" && ruleFile == "" {
		err := errors.New("error: input file and rule file must be provided")
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	if !util.FileExists(inputFile) {
		err := fmt.Errorf("error: input file %s doesn't exists", inputFile)
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	if !util.FileExists(ruleFile) {
		err := fmt.Errorf("error: rule file %s doesn't exists", ruleFile)
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func init() {
	cmds.RootCmd.AddCommand(kazaamCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// genCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	kazaamCmd.Flags().StringVarP(&inputFile, "in", "i", "", "json input file to process")
	kazaamCmd.Flags().StringVarP(&ruleFile, "rule", "r", "", "rule definition to apply")
	kazaamCmd.Flags().StringVarP(&outputFile, "out", "o", "", "output file with transformed data")
}
