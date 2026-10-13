// Editable settings groups and entity tab choices.
export const SETTINGS_METADATA = {
  "colors": [
    {
      "label": "Brand & Accent",
      "colors": [
        {
          "key": "brandPrimary",
          "label": "Primary"
        },
        {
          "key": "brandPrimaryHover",
          "label": "Primary Hover"
        },
        {
          "key": "brandSecondary",
          "label": "Secondary"
        },
        {
          "key": "brandAccent",
          "label": "Accent"
        }
      ]
    },
    {
      "label": "Backgrounds",
      "colors": [
        {
          "key": "bgPrimary",
          "label": "Primary BG"
        },
        {
          "key": "bgTertiary",
          "label": "Tertiary BG"
        }
      ]
    },
    {
      "label": "Text",
      "colors": [
        {
          "key": "textPrimary",
          "label": "Primary"
        },
        {
          "key": "textSecondary",
          "label": "Secondary"
        },
        {
          "key": "textTertiary",
          "label": "Muted"
        }
      ]
    },
    {
      "label": "Borders",
      "colors": [
        {
          "key": "borderMedium",
          "label": "Border"
        },
        {
          "key": "borderFocus",
          "label": "Focus Ring"
        }
      ]
    },
    {
      "label": "Status",
      "colors": [
        {
          "key": "colorSuccess",
          "label": "Success"
        },
        {
          "key": "colorWarning",
          "label": "Warning"
        },
        {
          "key": "colorError",
          "label": "Error / Danger"
        },
        {
          "key": "colorInfo",
          "label": "Info"
        }
      ]
    },
    {
      "label": "Kill / Loss",
      "colors": [
        {
          "key": "lossBg",
          "label": "Loss Background"
        },
        {
          "key": "lossHover",
          "label": "Loss Hover"
        },
        {
          "key": "lossBorder",
          "label": "Loss Border"
        }
      ]
    },
    {
      "label": "EVE Security Status",
      "colors": [
        {
          "key": "colorHighsec",
          "label": "Highsec"
        },
        {
          "key": "colorLowsec",
          "label": "Lowsec"
        },
        {
          "key": "colorNullsec",
          "label": "Nullsec"
        }
      ]
    },
    {
      "label": "Scrollbar",
      "colors": [
        {
          "key": "scrollbarThumb",
          "label": "Scrollbar Thumb"
        }
      ]
    },
    {
      "label": "Content",
      "colors": [
        {
          "key": "iskColor",
          "label": "ISK Values"
        },
        {
          "key": "npcColor",
          "label": "NPC Names"
        }
      ]
    },
    {
      "label": "Selection",
      "colors": [
        {
          "key": "selectionBg",
          "label": "Selection Background"
        },
        {
          "key": "selectionText",
          "label": "Selection Text"
        }
      ]
    }
  ],
  "tabs": [
    {
      "key": "character",
      "label": "Character",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "combined",
          "label": "Combined"
        },
        {
          "value": "kills",
          "label": "Kills"
        },
        {
          "value": "losses",
          "label": "Losses"
        },
        {
          "value": "history",
          "label": "History"
        },
        {
          "value": "achievements",
          "label": "Achievements"
        }
      ]
    },
    {
      "key": "corporation",
      "label": "Corporation",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "combined",
          "label": "Combined"
        },
        {
          "value": "kills",
          "label": "Kills"
        },
        {
          "value": "losses",
          "label": "Losses"
        },
        {
          "value": "members",
          "label": "Members"
        },
        {
          "value": "history",
          "label": "History"
        }
      ]
    },
    {
      "key": "alliance",
      "label": "Alliance",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "combined",
          "label": "Combined"
        },
        {
          "value": "kills",
          "label": "Kills"
        },
        {
          "value": "losses",
          "label": "Losses"
        },
        {
          "value": "corporations",
          "label": "Corporations"
        },
        {
          "value": "members",
          "label": "Members"
        }
      ]
    },
    {
      "key": "faction",
      "label": "Faction",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "combined",
          "label": "Combined"
        },
        {
          "value": "kills",
          "label": "Kills"
        },
        {
          "value": "losses",
          "label": "Losses"
        }
      ]
    },
    {
      "key": "item",
      "label": "Items / Ships / Groups",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "kills",
          "label": "Kills"
        }
      ]
    },
    {
      "key": "system",
      "label": "System",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "kills",
          "label": "Kills"
        }
      ]
    },
    {
      "key": "constellation",
      "label": "Constellation",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "kills",
          "label": "Kills"
        }
      ]
    },
    {
      "key": "region",
      "label": "Region",
      "options": [
        {
          "value": "dashboard",
          "label": "Dashboard"
        },
        {
          "value": "kills",
          "label": "Kills"
        }
      ]
    }
  ]
} as const
