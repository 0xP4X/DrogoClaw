package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

func RunLoadingSteps(stepLabels []string, execute func(int) error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	logo := HeaderBrandStyle.Render("  ___              _          _           _ _   ") + "\n" +
		HeaderBrandStyle.Render(" |   \\ _ _ __ _ _| |_ __ _  | | __ _ _ _| (_)__ ___ __") + "\n" +
		HeaderBrandStyle.Render(" | |) | '_/ _` |  _/ _` | | |/ _` | '_| | / _` \\ V / _ |") + "\n" +
		HeaderBrandStyle.Render(" |___/|_| \\__,_|\\__\\__,_| |_|\\__,_|_| |_|_|\\__,_\\_/ \\_,_|")

	fmt.Println()
	fmt.Println(logo)
	fmt.Println(HintDescStyle.Render("  Autonomous AI Security Testing Platform"))
	fmt.Println()

	for i, label := range stepLabels {
		if err := execute(i); err != nil {
			fmt.Printf("  %s  %s\n", ToolOutputErrorStyle.Render("✗"), HintDescStyle.Render(label))
			return err
		}

		fmt.Printf("  %s  %s\n", ToolOutputSuccessStyle.Render("✓"), SidebarValueStyle.Render(label))
		time.Sleep(60 * time.Millisecond)
	}

	fmt.Println()
	fmt.Println(ToolOutputSuccessStyle.Render("  Workspace ready."))
	fmt.Println()

	return nil
}
