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

	claw := []string{
		"                                :*%###*=",
		"                             .=#@@@@@#-:+*",
		"                        .:-=*%@@@@%+::*@%+",
		"                      :+%@@@@@%%%-:=#@#==-",
		"                     :+=-----+@%%::#*=+##-",
		"                  =#-%@@@@%+: +@-=* +%*-",
		"                =%@*=@%@*+++@= - =# =.",
		"              :#@%+*@%@+=%%@*.##*=",
		"           :=+-:=.-*%@==%@@* #%-=+:",
		"       .:=*@*=-.+%* +=.=-:%# -..=%@#:",
		"    =# -%#*+:::%@+=: -%@# :+#= * +@@*",
		"   :@#.=.+*+=-%@# +: @@%:* %#: **=+%@+",
		"  .#%==-    +@%=+#:.+@@*.+..   :*@*-*::#-",
		"  =%+#*  ***:-=##-  **#-%.       :+-:.#@#",
		"-%=.*=  -@@@ :-:   +== *#          .=-**=",
		"##=*+   -%@+--    #@@@::.         .*-=++=",
		"%++%=   #@%-@-    #%%*+#          :@%:%@-",
		"+##.   .+*+#*     +@@#@+           +++@=",
		" %#   :#*-:*      ==-=%-           **%-",
		" .%.  :@@=++      .*#+:          .##+",
		"  ..  .%%.%#      =@@-+.        .=-",
		"       *@=#.      .%%=+",
		"       .%#+        =@%=",
		"        -%%.        *@*",
		"         .#+         +%",
		"           =.         -:",
	}

	fmt.Println()
	for _, line := range claw {
		fmt.Println(HeaderBrandStyle.Render("  " + line))
	}
	fmt.Println()
	fmt.Println(HintDescStyle.Render("                    Autonomous AI Security Testing"))
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
