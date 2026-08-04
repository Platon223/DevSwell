package email

import "fmt"

// BrandedHTML wraps bodyHTML in DevSwell's actual visual identity (logo,
// colors matching the real light-theme web palette, consistent typography)
// instead of the ad-hoc unbranded markup each email previously had.
//
// Uses a table for the header and inline styles throughout, not flexbox/CSS
// variables — email clients (Outlook especially) have inconsistent CSS
// support, so this deliberately sticks to old-school, maximally-compatible
// HTML email patterns rather than what the web pages use.
//
// logoURL should be an absolute URL (e.g. baseURL+"/static/logo-mark.png")
// since email clients load images from real hosted URLs, not local paths —
// this only renders correctly once deployed with a real public baseURL,
// not against http://localhost during local testing.
func BrandedHTML(logoURL, heading, bodyHTML string) string {
	return fmt.Sprintf(`
<div style="background-color:#f6f8fa;padding:40px 20px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <div style="max-width:480px;margin:0 auto;background-color:#ffffff;border:1px solid #e5e5e5;border-radius:8px;padding:32px;">
    <table role="presentation" cellpadding="0" cellspacing="0" style="margin-bottom:24px;">
      <tr>
        <td style="padding-right:8px;"><img src="%s" alt="DevSwell" height="28" style="display:block;"></td>
        <td style="font-size:15px;font-weight:600;color:#171717;vertical-align:middle;">DevSwell</td>
      </tr>
    </table>
    <h2 style="font-size:18px;font-weight:600;color:#171717;margin:0 0 12px;">%s</h2>
    %s
    <p style="font-size:12px;color:#6b6b6b;margin:32px 0 0;padding-top:16px;border-top:1px solid #e5e5e5;">DevSwell &mdash; your tech stack, digested daily.</p>
  </div>
</div>`, logoURL, heading, bodyHTML)
}

// Button renders an email-safe call-to-action link (inline-block styling,
// no flexbox/grid needed) in the given accent color.
func Button(href, label, color string) string {
	return fmt.Sprintf(`<a href="%s" style="display:inline-block;padding:10px 20px;background-color:%s;color:#ffffff;text-decoration:none;border-radius:6px;font-weight:600;font-size:14px;">%s</a>`, href, color, label)
}
