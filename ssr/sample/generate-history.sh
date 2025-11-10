#!/bin/bash
# Script to generate sample git history for testing

cd "$(dirname "$0")"

# Initialize git if not already done
if [ ! -d .git ]; then
    git init
    git config user.name "Sample Data Generator"
    git config user.email "sample@example.com"
fi

# Commit 1: Initial creation of EVRY ASA (2020-01-15)
mkdir -p data/810/034/882
cat > data/810/034/882/enhet.json << 'ENHET1'
{
  "organisasjonsnummer": "810034882",
  "navn": "EVRY ASA",
  "organisasjonsform": {
    "kode": "ASA",
    "beskrivelse": "Allmennaksjeselskap"
  },
  "registreringsdatoEnhetsregisteret": "1995-08-09",
  "naeringskode1": {
    "kode": "62.010",
    "beskrivelse": "Programmeringstjenester"
  },
  "antallAnsatte": 3850,
  "forretningsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0275",
    "poststed": "OSLO",
    "adresse": ["Oksenøyveien 10"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  },
  "maalform": "Bokmål"
}
ENHET1

git add data/810/034/882/enhet.json
GIT_AUTHOR_DATE="2020-01-15T10:00:00" GIT_COMMITTER_DATE="2020-01-15T10:00:00" \
git commit -m "Initial import: EVRY ASA"

# Commit 2: Employee count update (2020-06-20)
cat > data/810/034/882/enhet.json << 'ENHET2'
{
  "organisasjonsnummer": "810034882",
  "navn": "EVRY ASA",
  "organisasjonsform": {
    "kode": "ASA",
    "beskrivelse": "Allmennaksjeselskap"
  },
  "registreringsdatoEnhetsregisteret": "1995-08-09",
  "naeringskode1": {
    "kode": "62.010",
    "beskrivelse": "Programmeringstjenester"
  },
  "antallAnsatte": 4100,
  "forretningsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0275",
    "poststed": "OSLO",
    "adresse": ["Oksenøyveien 10"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  },
  "maalform": "Bokmål"
}
ENHET2

git add data/810/034/882/enhet.json
GIT_AUTHOR_DATE="2020-06-20T14:30:00" GIT_COMMITTER_DATE="2020-06-20T14:30:00" \
git commit -m "Update employee count"

# Commit 3: Add MVA registration (2021-02-10)
cat > data/810/034/882/enhet.json << 'ENHET3'
{
  "organisasjonsnummer": "810034882",
  "navn": "EVRY ASA",
  "organisasjonsform": {
    "kode": "ASA",
    "beskrivelse": "Allmennaksjeselskap"
  },
  "registreringsdatoEnhetsregisteret": "1995-08-09",
  "registrertIMvaregisteret": true,
  "naeringskode1": {
    "kode": "62.010",
    "beskrivelse": "Programmeringstjenester"
  },
  "antallAnsatte": 4100,
  "forretningsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0275",
    "poststed": "OSLO",
    "adresse": ["Oksenøyveien 10"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  },
  "registrertIForetaksregisteret": true,
  "maalform": "Bokmål"
}
ENHET3

git add data/810/034/882/enhet.json
GIT_AUTHOR_DATE="2021-02-10T09:15:00" GIT_COMMITTER_DATE="2021-02-10T09:15:00" \
git commit -m "Add MVA registration and enterprise registry status"

# Commit 3b: Initial roller (2021-03-01)
cat > data/810/034/882/roller.json << 'ROLLER1'
{
  "rollegrupper": [
    {
      "type": {
        "kode": "STYR",
        "beskrivelse": "Styre"
      },
      "roller": [
        {
          "type": {
            "kode": "LEDE",
            "beskrivelse": "Styreleder"
          },
          "person": {
            "navn": {
              "fornavn": "Ole",
              "etternavn": "Hansen"
            },
            "fodselsdato": "1965-03-15"
          }
        },
        {
          "type": {
            "kode": "MEDL",
            "beskrivelse": "Styremedlem"
          },
          "person": {
            "navn": {
              "fornavn": "Kari",
              "etternavn": "Nilsen"
            },
            "fodselsdato": "1972-08-22"
          }
        }
      ]
    },
    {
      "type": {
        "kode": "DAGL",
        "beskrivelse": "Daglig ledelse"
      },
      "roller": [
        {
          "type": {
            "kode": "DAGL",
            "beskrivelse": "Daglig leder"
          },
          "person": {
            "navn": {
              "fornavn": "Per",
              "etternavn": "Andersen"
            },
            "fodselsdato": "1978-11-05"
          }
        }
      ]
    }
  ]
}
ROLLER1

git add data/810/034/882/roller.json
GIT_AUTHOR_DATE="2021-03-01T10:00:00" GIT_COMMITTER_DATE="2021-03-01T10:00:00" \
git commit -m "Add board and management roles"

# Commit 4: Employee growth and annual report (2022-03-25)
cat > data/810/034/882/enhet.json << 'ENHET4'
{
  "organisasjonsnummer": "810034882",
  "navn": "EVRY ASA",
  "organisasjonsform": {
    "kode": "ASA",
    "beskrivelse": "Allmennaksjeselskap"
  },
  "registreringsdatoEnhetsregisteret": "1995-08-09",
  "registrertIMvaregisteret": true,
  "naeringskode1": {
    "kode": "62.010",
    "beskrivelse": "Programmeringstjenester"
  },
  "antallAnsatte": 4234,
  "forretningsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0275",
    "poststed": "OSLO",
    "adresse": ["Oksenøyveien 10"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  },
  "institusjonellSektorkode": {
    "kode": "2100",
    "beskrivelse": "Private aksjeselskaper mv."
  },
  "registrertIForetaksregisteret": true,
  "sisteInnsendteAarsregnskap": "2021",
  "maalform": "Bokmål"
}
ENHET4

git add data/810/034/882/enhet.json
GIT_AUTHOR_DATE="2022-03-25T11:45:00" GIT_COMMITTER_DATE="2022-03-25T11:45:00" \
git commit -m "Employee growth and 2021 annual report submitted"

# Commit 4b: Add new board member (2022-06-20)
cat > data/810/034/882/roller.json << 'ROLLER2'
{
  "rollegrupper": [
    {
      "type": {
        "kode": "STYR",
        "beskrivelse": "Styre"
      },
      "roller": [
        {
          "type": {
            "kode": "LEDE",
            "beskrivelse": "Styreleder"
          },
          "person": {
            "navn": {
              "fornavn": "Ole",
              "etternavn": "Hansen"
            },
            "fodselsdato": "1965-03-15"
          }
        },
        {
          "type": {
            "kode": "MEDL",
            "beskrivelse": "Styremedlem"
          },
          "person": {
            "navn": {
              "fornavn": "Kari",
              "etternavn": "Nilsen"
            },
            "fodselsdato": "1972-08-22"
          }
        },
        {
          "type": {
            "kode": "MEDL",
            "beskrivelse": "Styremedlem"
          },
          "person": {
            "navn": {
              "fornavn": "Lars",
              "etternavn": "Johansen"
            },
            "fodselsdato": "1980-04-12"
          }
        }
      ]
    },
    {
      "type": {
        "kode": "DAGL",
        "beskrivelse": "Daglig ledelse"
      },
      "roller": [
        {
          "type": {
            "kode": "DAGL",
            "beskrivelse": "Daglig leder"
          },
          "person": {
            "navn": {
              "fornavn": "Per",
              "etternavn": "Andersen"
            },
            "fodselsdato": "1978-11-05"
          }
        }
      ]
    }
  ]
}
ROLLER2

git add data/810/034/882/roller.json
GIT_AUTHOR_DATE="2022-06-20T14:00:00" GIT_COMMITTER_DATE="2022-06-20T14:00:00" \
git commit -m "New board member Lars Johansen added"

# Commit 5: Latest annual report (2023-04-12)
cat > data/810/034/882/enhet.json << 'ENHET5'
{
  "organisasjonsnummer": "810034882",
  "navn": "EVRY ASA",
  "organisasjonsform": {
    "kode": "ASA",
    "beskrivelse": "Allmennaksjeselskap"
  },
  "registreringsdatoEnhetsregisteret": "1995-08-09",
  "registrertIMvaregisteret": true,
  "naeringskode1": {
    "kode": "62.010",
    "beskrivelse": "Programmeringstjenester"
  },
  "antallAnsatte": 4234,
  "forretningsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0275",
    "poststed": "OSLO",
    "adresse": ["Oksenøyveien 10"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  },
  "institusjonellSektorkode": {
    "kode": "2100",
    "beskrivelse": "Private aksjeselskaper mv."
  },
  "registrertIForetaksregisteret": true,
  "registrertIStiftelsesregisteret": false,
  "registrertIFrivillighetsregisteret": false,
  "sisteInnsendteAarsregnskap": "2022",
  "konkurs": false,
  "underAvvikling": false,
  "underTvangsavviklingEllerTvangsopplosning": false,
  "maalform": "Bokmål"
}
ENHET5

git add data/810/034/882/enhet.json
GIT_AUTHOR_DATE="2023-04-12T08:20:00" GIT_COMMITTER_DATE="2023-04-12T08:20:00" \
git commit -m "Update 2022 annual report and registry statuses"

# Commit 6: Add underenhet (2021-05-10)
mkdir -p data/923/456/789
cat > data/923/456/789/underenhet.json << 'UNDER1'
{
  "organisasjonsnummer": "923456789",
  "navn": "OSLO CONSULTING AVDELING",
  "overordnetEnhet": "810034882",
  "organisasjonsform": {
    "kode": "BEDR",
    "beskrivelse": "Bedrift"
  },
  "registreringsdatoEnhetsregisteret": "2010-03-15",
  "naeringskode1": {
    "kode": "62.020",
    "beskrivelse": "Konsulent tjenester tilknyttet informasjonsteknologi"
  },
  "antallAnsatte": 32,
  "beliggenhetsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0150",
    "poststed": "OSLO",
    "adresse": ["Storgata 123"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  }
}
UNDER1

git add data/923/456/789/underenhet.json
GIT_AUTHOR_DATE="2021-05-10T13:00:00" GIT_COMMITTER_DATE="2021-05-10T13:00:00" \
git commit -m "Initial import: Oslo Consulting Avdeling"

# Commit 7: Underenhet employee growth (2022-08-15)
cat > data/923/456/789/underenhet.json << 'UNDER2'
{
  "organisasjonsnummer": "923456789",
  "navn": "OSLO CONSULTING AVDELING",
  "overordnetEnhet": "810034882",
  "organisasjonsform": {
    "kode": "BEDR",
    "beskrivelse": "Bedrift"
  },
  "registreringsdatoEnhetsregisteret": "2010-03-15",
  "registrertIMvaregisteret": false,
  "naeringskode1": {
    "kode": "62.020",
    "beskrivelse": "Konsulent tjenester tilknyttet informasjonsteknologi"
  },
  "antallAnsatte": 45,
  "beliggenhetsadresse": {
    "land": "Norge",
    "landkode": "NO",
    "postnummer": "0150",
    "poststed": "OSLO",
    "adresse": ["Storgata 123"],
    "kommune": "OSLO",
    "kommunenummer": "0301"
  }
}
UNDER2

git add data/923/456/789/underenhet.json
GIT_AUTHOR_DATE="2022-08-15T16:30:00" GIT_COMMITTER_DATE="2022-08-15T16:30:00" \
git commit -m "Employee count update for Oslo office"

echo ""
echo "Sample git history created with 9 commits:"
git log --oneline --all
echo ""
echo "Ready to test gen-history tool!"
