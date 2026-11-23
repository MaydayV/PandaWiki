package share

import (
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/utils"
)

const shareLocaleCookieName = "pw_locale"

func shareQueryAndCookieLocale(c echo.Context) (queryLang, headerXLang, cookieLang, acceptLang string) {
	if c == nil || c.Request() == nil {
		return "", "", "", ""
	}
	queryLang = c.QueryParam("lang")
	if queryLang == "" {
		queryLang = c.QueryParam("language")
	}
	headerXLang = c.Request().Header.Get("X-Lang")
	if cookie, err := c.Cookie(shareLocaleCookieName); err == nil && cookie != nil {
		cookieLang = cookie.Value
	}
	acceptLang = c.Request().Header.Get("Accept-Language")
	return queryLang, headerXLang, cookieLang, acceptLang
}

func extractShareRequestLanguage(c echo.Context) (string, bool) {
	q, x, ck, al := shareQueryAndCookieLocale(c)
	return utils.ResolveMatchedLocaleFromHTTP(q, x, ck, al)
}

func resolveShareLocale(c echo.Context, fallback string) string {
	q, x, ck, al := shareQueryAndCookieLocale(c)
	return utils.ResolveLocaleFromHTTP(q, x, ck, al, fallback)
}

func shareMsg(locale, zh, en string) string {
	if utils.IsChineseLocale(locale) {
		return zh
	}
	return en
}
