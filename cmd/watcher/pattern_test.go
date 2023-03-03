package main

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func init() {
	showAlert = func(logger *logrus.Logger, title, msg string) error {
		return fmt.Errorf("%s: %s", title, msg)
	}
}

func TestFolderAttributes(t *testing.T) {
	testCases := []struct {
		FolderName    string
		WantFrameType string
		WantFrameSize string
	}{
		{"framed 2pc 11x14", "framed 2pc", "11x14"},
		{"framed 2pc 12x12", "framed 2pc", "12x12"},
		{"framed 3pc 11x14", "framed 3pc", "11x14"},
		{"framed 3pc 12x12", "framed 3pc", "12x12"},
		{"framed 4pc 11x14", "framed 4pc", "11x14"},
		{"framed 4pc 12x12", "framed 4pc", "12x12"},
		{"framed 9pc 11x14", "framed 9pc", "11x14"},
		{"framed 9pc 12x12", "framed 9pc", "12x12"},
		{"framed 11x14", "framed", "11x14"},
		{"framed 12x12", "framed", "12x12"},
		{"gray framed 2pc 12x12", "gray framed 2pc", "12x12"},
		{"gray framed 3pc 11x14", "gray framed 3pc", "11x14"},
		{"gray framed 4pc 11x14", "gray framed 4pc", "11x14"},
		{"gray framed 4pc 12x12", "gray framed 4pc", "12x12"},
		{"gray framed 9pc 11x14", "gray framed 9pc", "11x14"},
		{"gray framed 9pc 12x12", "gray framed 9pc", "12x12"},
		{"gray framed 11x14", "gray framed", "11x14"},
		{"gray framed 12x12", "gray framed", "12x12"},
		{"white framed 2pc 11x14", "white framed 2pc", "11x14"},
		{"white framed 2pc 12x12", "white framed 2pc", "12x12"},
		{"white framed 3pc 11x14", "white framed 3pc", "11x14"},
		{"white framed 4pc 12x12", "white framed 4pc", "12x12"},
		{"white framed 4pc 11x14", "white framed 4pc", "11x14"},
		{"white framed 9pc 11x14", "white framed 9pc", "11x14"},
		{"white framed 11x14", "white framed", "11x14"},
		{"white framed 12x12", "white framed", "12x12"},
		{"wood 2pc 7x17", "wood 2pc", "7x17"},
		{"wood 2pc 10x15", "wood 2pc", "10x15"},
		{"wood 2pc 12x12", "wood 2pc", "12x12"},
		{"wood 2pc 13x19", "wood 2pc", "13x19"},
		{"wood 3pc 7x17", "wood 3pc", "7x17"},
		{"wood 3pc 10x15", "wood 3pc", "10x15"},
		{"wood 3pc 11x17", "wood 3pc", "11x17"},
		{"wood 3pc 12x12", "wood 3pc", "12x12"},
		{"wood 3pc 13x19", "wood 3pc", "13x19"},
		{"wood 4pc 10x15", "wood 4pc", "10x15"},
		{"wood 4pc 12x12", "wood 4pc", "12x12"},
		{"wood 12x12", "wood", "12x12"},
		{"wood crx 12x12", "wood crx", "12x12"},
		{"wood horz 7x17", "wood horz", "7x17"},
		{"wood horz 10x15", "wood horz", "10x15"},
		{"wood horz 13x19", "wood horz", "13x19"},
		{"wood vert 7x17", "wood vert", "7x17"},
		{"wood vert 10x15", "wood vert", "10x15"},
		{"wood vert 13x19", "wood vert", "13x19"},
		{"10x21", "", "10x21"},
		{"10x24", "", "10x24"},
		{"10x24 black framed", "black framed", "10x24"},
		{"10x24 white framed", "white framed", "10x24"},
		{"11x14", "", "11x14"},
		{"12x12", "", "12x12"},
		{"13x30", "", "13x30"},
		{"13x30 black framed", "black framed", "13x30"},
		{"13x30 gray framed", "gray framed", "13x30"},
		{"13x30 white framed", "white framed", "13x30"},
		{"16x20 floating black framed", "floating black framed", "16x20"},
		{"16x20 floating gold framed", "floating gold framed", "16x20"},
		{"16x20 floating gray framed", "floating gray framed", "16x20"},
		{"16x20 gray framed", "gray framed", "16x20"},
		{"16x20 white framed", "white framed", "16x20"},
		{"16x24", "", "16x24"},
		{"17x17", "", "17x17"},
		{"17x17 black framed", "black framed", "17x17"},
		{"17x17 gray framed", "gray framed", "17x17"},
		{"17x17 white framed", "white framed", "17x17"},
		{"18x18", "", "18x18"},
		{"20x48", "", "20x48"},
		{"24x24", "", "24x24"},
		{"24x24 black framed", "black framed", "24x24"},
		{"24x24 white framed", "white framed", "24x24"},
		{"24x24 gray framed", "gray framed", "24x24"},
		{"24x30", "", "24x30"},
		{"24x30 black framed", "black framed", "24x30"},
		{"24x30 floating black framed", "floating black framed", "24x30"},
		{"24x30 floating gold framed", "floating gold framed", "24x30"},
		{"24x30 floating gray framed", "floating gray framed", "24x30"},
		{"24x30 white framed", "white framed", "24x30"},
		{"30x30", "", "30x30"},
		{"30x40", "", "30x40"},
		{"36x36", "", "36x36"},
		{"36x48", "", "36x48"},
	}

	logger := logrus.New()

	folderAttrPatterns := []string{
		`^(?P<frame_size>\d+x\d+)$`,
		`^(?P<frame_size>\d+x\d+) (?P<frame_type>(floating )?((white|gray|black|gold) )?framed)$`,
		`^(?P<frame_type>(floating )?((white|gray|black|gold) )?framed( \d+pc)?) (?P<frame_size>\d+x\d+)$`,
		`^(?P<frame_type>(wood|wood horz|wood vert|wood crx|framed)( \d+pc)?) (?P<frame_size>\d+x\d+)$`,
	}

	for _, tc := range testCases {
		attrs, err := getFolderAttributes(logger, tc.FolderName, folderAttrPatterns)
		if err != nil {
			t.Errorf("got unexpected error when retrieving attributes from %q, want=nil, got=%v", tc.FolderName, err)
		}

		gotFrameSize, ok := attrs[attrFrameSize]
		if !ok || gotFrameSize != tc.WantFrameSize {
			t.Errorf("got unexpected frame size for %q, want=%s, got=%s", tc.FolderName, tc.WantFrameSize, gotFrameSize)
		}

		gotFrameType, ok := attrs[attrFrameType]
		if !ok || gotFrameType != tc.WantFrameType {
			t.Errorf("got unexpected frame type for %q, want=%s, got=%s", tc.FolderName, tc.WantFrameType, gotFrameType)
		}
	}
}
