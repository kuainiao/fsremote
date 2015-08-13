package main

type hi_session_protocol struct {
	Domain string                 `json:"domain"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
	text   string
}

func when_session_protocol(p hi_session_protocol) sm_semantic {
	var v sm_semantic
	return v
}

//WAITING
type hi_waiting struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	OnCancel string `json:"onCancel"` //[{"domain":"DOMAIN_LOCAL","confirm":"cancel","message":"取消"}]
}
type hi_on_cancel struct {
	Domain  string `json:"domain"`
	Confirm string `json:"confirm"`
	Message string `json:"message"`
}

type hi_mutiple_contacts struct {
}

type hi_multiple_numbers struct {
}

type hi_confirm_call struct {
}

type hi_input_contact struct {
}

type hi_call_ok struct {
}

type hi_input_freetext_sms struct {
}

type hi_sms_ok struct {
}

type hi_contact_show struct {
}

type hi_reminder_show struct {
}

type hi_reminder_ok struct {
}

type hi_app_launch struct {
}

type hi_app_uninstall struct {
}

type hi_music_show struct {
}

type hi_channel_prog_list struct {
}

type hi_prog_search_result struct {
}

type hi_prog_recommend struct {
}

type hi_web_show struct {
}

type hi_poi_show struct {
}

type hi_position_show struct {
}

type hi_route_show struct {
}

type hi_stock_show struct {
}

type hi_translation_show struct {
}

type hi_setting struct {
}

type hi_input_freetext_weibo struct {
}

type hi_confirm_weibo struct {
}

type hi_weather_show struct {
}

type hi_multiple_app struct {
}

type hi_multiple_show struct {
}

type hi_contact_add struct {
}

type hi_sms_read struct {
}

type hi_alarm_show struct {
}

type hi_alarm_ok struct {
}

type hi_talk_show struct {
}