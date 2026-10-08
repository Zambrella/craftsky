package moderation

type ReportGroup string

const (
	GroupChildSafety                   ReportGroup = "childSafety"
	GroupSexualIntimate                ReportGroup = "sexualIntimateContent"
	GroupThreatsViolenceSelfHarm       ReportGroup = "threatsViolenceSelfHarm"
	GroupHarassmentHateStalkingPrivacy ReportGroup = "harassmentHateStalkingPrivacy"
	GroupFraudScamsImpersonation       ReportGroup = "fraudScamsImpersonation"
	GroupSpamCommercialOffTopic        ReportGroup = "spamCommercialOffTopic"
	GroupIntellectualProperty          ReportGroup = "intellectualProperty"
	GroupOther                         ReportGroup = "other"
)

type Allegation string

const (
	AllegationChildSexualExploitation Allegation = "childSexualExploitation"
	AllegationGrooming                Allegation = "grooming"
	AllegationChildAbuseMaterial      Allegation = "childAbuseMaterial"
	AllegationIntimateImageAbuse      Allegation = "intimateImageAbuse"
	AllegationSexualExploitation      Allegation = "sexualExploitation"
	AllegationAdultContent            Allegation = "adultContent"
	AllegationGraphicContent          Allegation = "graphicContent"
	AllegationImmediateDanger         Allegation = "immediateDanger"
	AllegationCredibleThreat          Allegation = "credibleThreat"
	AllegationTerrorism               Allegation = "terrorism"
	AllegationViolence                Allegation = "violence"
	AllegationSelfHarm                Allegation = "selfHarm"
	AllegationHarassment              Allegation = "harassment"
	AllegationHate                    Allegation = "hate"
	AllegationStalking                Allegation = "stalking"
	AllegationDoxxing                 Allegation = "doxxing"
	AllegationBlockEvasion            Allegation = "blockEvasion"
	AllegationPrivacy                 Allegation = "privacy"
	AllegationFraud                   Allegation = "fraud"
	AllegationScam                    Allegation = "scam"
	AllegationPhishing                Allegation = "phishing"
	AllegationImpersonation           Allegation = "impersonation"
	AllegationCounterfeit             Allegation = "counterfeit"
	AllegationSpam                    Allegation = "spam"
	AllegationPlatformManipulation    Allegation = "platformManipulation"
	AllegationMisleading              Allegation = "misleading"
	AllegationSuspectedAIGenerated    Allegation = "suspectedAiGenerated"
	AllegationOffTopic                Allegation = "offTopic"
	AllegationIntellectualProperty    Allegation = "intellectualProperty"
	AllegationOther                   Allegation = "other"
)

type ReportDestination string

const (
	DestinationInApp           ReportDestination = "inApp"
	DestinationSpecialistEmail ReportDestination = "specialistEmail"
)

type LegalClassification string

const (
	LegalUnassessed              LegalClassification = "unassessed"
	LegalChildSexualExploitation LegalClassification = "childSexualExploitation"
	LegalIntimateImageAbuse      LegalClassification = "intimateImageAbuse"
	LegalCredibleThreat          LegalClassification = "credibleThreat"
	LegalOtherIllegalContent     LegalClassification = "otherIllegalContent"
)

type AllegationMapping struct {
	Group               ReportGroup
	DecisionReason      Reason
	LegalClassification LegalClassification
	UserSafeLabel       string
	Destination         ReportDestination
}

func ApprovedAllegationMappings() map[Allegation]AllegationMapping {
	return map[Allegation]AllegationMapping{
		AllegationChildSexualExploitation: {GroupChildSafety, ReasonChildSafety, LegalUnassessed, "Child sexual exploitation", DestinationInApp},
		AllegationGrooming:                {GroupChildSafety, ReasonChildSafety, LegalUnassessed, "Grooming", DestinationInApp},
		AllegationChildAbuseMaterial:      {GroupChildSafety, ReasonChildSafety, LegalUnassessed, "Child sexual abuse material", DestinationInApp},
		AllegationIntimateImageAbuse:      {GroupSexualIntimate, ReasonSexualViolation, LegalUnassessed, "Intimate image abuse", DestinationInApp},
		AllegationSexualExploitation:      {GroupSexualIntimate, ReasonSexualViolation, LegalUnassessed, "Sexual exploitation", DestinationInApp},
		AllegationAdultContent:            {GroupSexualIntimate, ReasonAdultOrGraphic, LegalUnassessed, "Adult content", DestinationInApp},
		AllegationGraphicContent:          {GroupSexualIntimate, ReasonAdultOrGraphic, LegalUnassessed, "Graphic content", DestinationInApp},
		AllegationImmediateDanger:         {GroupThreatsViolenceSelfHarm, ReasonThreatOrViolence, LegalUnassessed, "Someone is in immediate danger", DestinationInApp},
		AllegationCredibleThreat:          {GroupThreatsViolenceSelfHarm, ReasonThreatOrViolence, LegalUnassessed, "Credible threat", DestinationInApp},
		AllegationTerrorism:               {GroupThreatsViolenceSelfHarm, ReasonThreatOrViolence, LegalUnassessed, "Terrorism", DestinationInApp},
		AllegationViolence:                {GroupThreatsViolenceSelfHarm, ReasonThreatOrViolence, LegalUnassessed, "Violence", DestinationInApp},
		AllegationSelfHarm:                {GroupThreatsViolenceSelfHarm, ReasonSelfHarm, LegalUnassessed, "Self-harm concern", DestinationInApp},
		AllegationHarassment:              {GroupHarassmentHateStalkingPrivacy, ReasonHarassment, LegalUnassessed, "Harassment", DestinationInApp},
		AllegationHate:                    {GroupHarassmentHateStalkingPrivacy, ReasonHate, LegalUnassessed, "Hate", DestinationInApp},
		AllegationStalking:                {GroupHarassmentHateStalkingPrivacy, ReasonStalkingOrPrivacy, LegalUnassessed, "Stalking", DestinationInApp},
		AllegationDoxxing:                 {GroupHarassmentHateStalkingPrivacy, ReasonStalkingOrPrivacy, LegalUnassessed, "Doxxing", DestinationInApp},
		AllegationBlockEvasion:            {GroupHarassmentHateStalkingPrivacy, ReasonStalkingOrPrivacy, LegalUnassessed, "Block evasion", DestinationInApp},
		AllegationPrivacy:                 {GroupHarassmentHateStalkingPrivacy, ReasonStalkingOrPrivacy, LegalUnassessed, "Privacy violation", DestinationInApp},
		AllegationFraud:                   {GroupFraudScamsImpersonation, ReasonFraudOrScam, LegalUnassessed, "Fraud", DestinationInApp},
		AllegationScam:                    {GroupFraudScamsImpersonation, ReasonFraudOrScam, LegalUnassessed, "Scam", DestinationInApp},
		AllegationPhishing:                {GroupFraudScamsImpersonation, ReasonFraudOrScam, LegalUnassessed, "Phishing", DestinationInApp},
		AllegationImpersonation:           {GroupFraudScamsImpersonation, ReasonImpersonation, LegalUnassessed, "Impersonation", DestinationInApp},
		AllegationCounterfeit:             {GroupFraudScamsImpersonation, ReasonFraudOrScam, LegalUnassessed, "Counterfeit goods", DestinationInApp},
		AllegationSpam:                    {GroupSpamCommercialOffTopic, ReasonSpam, LegalUnassessed, "Spam", DestinationInApp},
		AllegationPlatformManipulation:    {GroupSpamCommercialOffTopic, ReasonMisleadingCommercial, LegalUnassessed, "Platform manipulation", DestinationInApp},
		AllegationMisleading:              {GroupSpamCommercialOffTopic, ReasonMisleading, LegalUnassessed, "Misleading content", DestinationInApp},
		AllegationSuspectedAIGenerated:    {GroupSpamCommercialOffTopic, ReasonSuspectedAIGenerated, LegalUnassessed, "Suspected AI-generated content", DestinationInApp},
		AllegationOffTopic:                {GroupSpamCommercialOffTopic, ReasonOffTopic, LegalUnassessed, "Off-topic content", DestinationInApp},
		AllegationIntellectualProperty:    {GroupIntellectualProperty, "", LegalUnassessed, "Copyright or trade mark", DestinationSpecialistEmail},
		AllegationOther:                   {GroupOther, ReasonOther, LegalUnassessed, "Another concern", DestinationInApp},
	}
}
