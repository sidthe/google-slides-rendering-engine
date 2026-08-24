package pptx

import (
	"fmt"

	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// ThemeXML renders the OOXML theme part for t.
func ThemeXML(t ir.Theme) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="%s"><a:themeElements>
<a:clrScheme name="c"><a:dk1><a:srgbClr val="%s"/></a:dk1><a:lt1><a:srgbClr val="%s"/></a:lt1><a:dk2><a:srgbClr val="%s"/></a:dk2><a:lt2><a:srgbClr val="%s"/></a:lt2><a:accent1><a:srgbClr val="%s"/></a:accent1><a:accent2><a:srgbClr val="%s"/></a:accent2><a:accent3><a:srgbClr val="%s"/></a:accent3><a:accent4><a:srgbClr val="%s"/></a:accent4><a:accent5><a:srgbClr val="%s"/></a:accent5><a:accent6><a:srgbClr val="%s"/></a:accent6><a:hlink><a:srgbClr val="%s"/></a:hlink><a:folHlink><a:srgbClr val="%s"/></a:folHlink></a:clrScheme>
<a:fontScheme name="f"><a:majorFont><a:latin typeface="%s"/><a:ea typeface=""/><a:cs typeface=""/></a:majorFont><a:minorFont><a:latin typeface="%s"/><a:ea typeface=""/><a:cs typeface=""/></a:minorFont></a:fontScheme>
<a:fmtScheme name="s"><a:fillStyleLst><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:fillStyleLst>
<a:lnStyleLst><a:ln w="6350"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln><a:ln w="12700"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln><a:ln w="19050"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln></a:lnStyleLst>
<a:effectStyleLst><a:effectStyle><a:effectLst/></a:effectStyle><a:effectStyle><a:effectLst/></a:effectStyle><a:effectStyle><a:effectLst/></a:effectStyle></a:effectStyleLst>
<a:bgFillStyleLst><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:bgFillStyleLst></a:fmtScheme>
</a:themeElements></a:theme>`,
		esc(t.Name), t.Dark1, t.Light1, t.Dark2, t.Light2,
		t.Accents[0], t.Accents[1], t.Accents[2], t.Accents[3], t.Accents[4], t.Accents[5],
		t.Hyperlink, t.FollowedLink, esc(t.Font), esc(t.Font))
}
