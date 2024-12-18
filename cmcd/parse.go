package cmcd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"errors"
)

func parseInfo(tokens map[string]string) (Info, error) {
	var (
		info Info
		err  error
		errs error
	)
	info.Request, err = parseRequest(tokens)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("request: %w", err))
	}
	info.Object, err = parseObject(tokens)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("object: %w", err))
	}
	info.Status, err = parseStatus(tokens)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("status: %w", err))
	}
	info.Session, err = parseSession(tokens)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("session: %w", err))
	}
	if custom := parseCustom(tokens); custom != nil {
		info.Custom = custom
	}
	return info, errs
}

func parseRequest(attrs map[string]string) (Request, error) {
	var (
		req  Request
		errs error
	)
	for k, v := range attrs {
		switch k {
		case "bl":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse buffer length: %w", err))
				continue
			}
			req.BufLength = time.Duration(i) * time.Millisecond
		case "dl":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse deadline: %w", err))
				continue
			}
			req.Deadline = time.Duration(i) * time.Millisecond
		case "mtp":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse throughput: %w", err))
				continue
			}
			req.Throughput = i
		case "nor":
			dec, err := url.QueryUnescape(strings.Trim(v, `"`))
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("decode next object request: %w", err))
				continue
			}
			req.Next = dec
		case "nrr":
			rg, err := parseRange(strings.Trim(v, `"`))
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse next range: %w", err))
				continue
			}
			req.NextRange = rg
		case "su":
			req.Startup = true
		}
	}
	return req, errs
}

func parseObject(attrs map[string]string) (Object, error) {
	var (
		obj  Object
		errs error
	)
	for k, v := range attrs {
		if v == "" {
			continue // stray comma, perhaps at end of line. ignore
		}
		switch k {
		case "br":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse bitrate: %w", err))
				continue
			}
			obj.Bitrate = i
		case "d":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse duration: %w", err))
				continue
			}
			obj.Duration = time.Duration(i) * time.Millisecond
		case "ot":
			// TODO(otl): validate value
			obj.Type = ObjectType(v)
		case "tb":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse top bitrate: %w", err))
				continue
			}
			obj.TopBitrate = i
		}
	}
	return obj, errs
}

func parseStatus(attrs map[string]string) (Status, error) {
	var (
		stat Status
		errs error
	)

	for k, v := range attrs {
		switch k {
		case "bs":
			stat.Starved = true
		case "rtp":
			i, err := strconv.Atoi(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse max throughput: %w", err))
				continue
			}
			stat.MaxThroughput = i
		}
	}
	return stat, errs
}

func parseSession(attrs map[string]string) (Session, error) {
	var (
		ses  Session
		errs error
	)
	for k, v := range attrs {
		switch k {
		case "sid":
			ses.ID = strings.Trim(v, `"`)
		case "st":
			ses.StreamType = "l"
		case "cid":
			ses.ContentID = strings.Trim(v, `"`)
		case "pr":
			i, err := strconv.ParseFloat(v, 32)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("parse play rate: %w", err))
				continue
			}
			ses.PlayRate = PlayRate(i)
		case "sf":
			if len(v) != 1 {
				errs = errors.Join(errs, fmt.Errorf("stream format: %s is not a single character", v))
				continue
			}
			c := StreamFormat([]byte(v)[0])
			switch c {
			case FormatDASH, FormatHLS, FormatSmooth, FormatOther:
				ses.Format = c
			default:
				errs = errors.Join(errs, fmt.Errorf("stream format: unknown format %c", c))
				continue
			}
		}
	}
	// If we didn't see playrate, we must set it to realtime as
	// only values other than realtime should be transmitted. See
	// CTA=5004 page 10.
	if _, ok := attrs["pr"]; !ok {
		ses.PlayRate = RealTime
	}
	return ses, errs
}

func parseCustom(attrs map[string]string) map[string]any {
	m := make(map[string]any)
	for k, v := range attrs {
		switch k {
		case "bl", "dl", "mtp", "nor", "nrr", "su":
			continue // Request keys
		case "br", "d", "ot", "tb":
			continue // Object keys
		case "bs", "rtp":
			continue // Status keys
		case "sid", "st", "cid", "pr", "sf":
			continue // Session keys
		}
		if v == "" {
			m[k] = true
		} else if i, err := strconv.Atoi(v); err == nil {
			m[k] = i
		} else {
			m[k] = strings.Trim(v, `"`)
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func lex(s string) map[string]string {
	m := make(map[string]string)
	s = clean(s)
	for _, attr := range strings.Split(s, ",") {
		name, val, _ := strings.Cut(attr, "=")
		m[name] = val
	}
	return m
}

// clean removes stray commas. Trailing commas are technically valid
// but we remove them to simplify parsing.
func clean(s string) string { return strings.Trim(s, ",") }
