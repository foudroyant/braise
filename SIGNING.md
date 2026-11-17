# Signature et vérification des binaires

## En bref

Les binaires de Braise **ne sont pas encore signés**. Windows SmartScreen et macOS Gatekeeper affichent donc un avertissement de sécurité au premier lancement. Ce n'est pas la preuve d'un virus : c'est le comportement normal d'un exécutable dont l'éditeur n'est pas encore identifié par le système.

La signature de code ne rend pas un programme « sûr » à elle seule. Elle apporte deux choses :

- **l'identité de l'éditeur** : le certificat dit qui a signé le fichier ;
- **l'intégrité** : si le fichier est modifié après signature, la signature devient invalide.

Elle ne remplace ni la vérification du code source, ni les empreintes SHA-256, ni un antivirus.

## Pourquoi l'avertissement apparaît

| Système | Comportement sans signature | Ce qui le fait disparaître |
| --- | --- | --- |
| Windows | SmartScreen affiche « Windows a protégé votre PC » | Signature Authenticode + réputation du certificat |
| macOS | Gatekeeper bloque : « développeur non identifié » | Certificat Developer ID + notarisation Apple |
| Linux | Pas de contrôle centralisé | Paquet signé par la distribution, ou empreintes/GPG |

Un certificat **auto-signé** ne règle pas le problème : il est inconnu des systèmes des utilisateurs et produit souvent un blocage encore plus fort. Il ne sert qu'au développement ou à un parc d'entreprise qui déploie lui-même le certificat.

## Options et coûts

Les prix ci-dessous sont des ordres de grandeur 2026. Ils changent selon le certificat, la durée et le pays.

| Plateforme | Option | Coût indicatif | Remarques |
| --- | --- | --- | --- |
| Windows | **SignPath Foundation** | Gratuit pour l'open source | Certificat OV géré par SignPath ; le build doit venir du dépôt public |
| Windows | Certificat OV classique | 150–300 $/an | Clé privée sur HSM ou token matériel depuis 2023 |
| Windows | Azure Artifact Signing (ex-Trusted Signing) | ~9,99 $/mois | Organisations : USA, Canada, UE, Royaume-Uni. Individus : USA et Canada uniquement |
| Windows | Certificat EV | 400 $+/an | Plus d'avantage immédiat contre SmartScreen depuis 2024 |
| macOS | Apple Developer Program | 99 $/an | Developer ID + notarisation ; obligatoire hors App Store |
| Linux | SHA-256, GPG ou cosign | Gratuit | Pas de Gatekeeper central |

### Windows : ce qu'il faut retenir

- **SignPath Foundation** est la meilleure option gratuite pour un projet open source. Le projet doit avoir une **licence approuvée par l'OSI**, être public, maintenu, déjà publié, et accepter une politique de signature.
- Un certificat **OV** classique fonctionne partout, mais demande une validation d'identité et un support HSM/token. C'est l'option réaliste si le projet reste sans licence open source.
- **Azure Artifact Signing** n'est pas disponible pour un développeur individuel basé au Congo. Il faudrait une entité dans un pays éligible.
- **EV** n'apporte plus de contournement immédiat de SmartScreen. Inutile de payer le supplément pour cela.
- Une signature neuve peut **toujours** afficher un avertissement SmartScreen tant que la réputation du certificat n'est pas établie. La réputation se construit avec des releases régulières signées par la même identité.

### macOS : ce qu'il faut retenir

- Il faut un **Apple Developer Program** payant (99 $/an). Un compte gratuit ne délivre pas les certificats Developer ID.
- Un binaire en ligne de commande se signe avec un certificat **Developer ID Application**, mais il ne se notarise pas directement. Il faut le placer dans un **`.pkg`** (ou un `.dmg`/`.app`) signé avec un certificat **Developer ID Installer**, puis notarier et agrafer (`staple`) ce paquet.
- Sans notarisation, même un binaire signé peut être bloqué par Gatekeeper.
- Il n'existe pas d'équivalent gratuit crédible pour distribuer hors App Store.

### Linux

Linux ne demande pas de certificat central pour exécuter un binaire. Les distributions et les utilisateurs vérifient plutôt :

- les empreintes SHA-256 ;
- une signature GPG ou Sigstore/cosign ;
- l'intégration dans un dépôt de distribution (Flathub, dépôt APT, etc.).

## Recommandation pour Braise

1. **Windows** : viser SignPath Foundation après avoir choisi une licence OSI. Sinon, acheter un certificat OV.
2. **macOS** : Apple Developer Program, puis distribution en `.pkg` signé et notarié.
3. **Linux** : publier `checksums.txt` et, à terme, une signature cosign ou GPG.
4. **Toutes plateformes** : afficher les empreintes et les commandes de vérification sur la page de téléchargement.

La release `v0.1.0` restera non signée. Une signature s'applique à un nouveau build : prévoir une `v0.1.1` ou une version suivante.

## Windows avec SignPath Foundation

### Prérequis

- dépôt public et maintenu ;
- **licence OSI** pour tout le code (MIT, Apache-2.0, GPL-3.0, etc.) ;
- projet déjà publié et documenté ;
- politique de signature visible sur la page d'accueil ou la page de téléchargement ;
- authentification à deux facteurs sur GitHub et SignPath ;
- rôles de committer, reviewer et approbateur ;
- approbation manuelle de chaque release.

**Braise n'a pas encore de licence.** C'est le point bloquant pour SignPath : un dépôt public sans licence n'est pas juridiquement open source. Il faut d'abord ajouter un fichier `LICENSE`.

### Procédure

1. Choisir une licence OSI et ajouter `LICENSE` à la racine.
2. Ajouter une section **Politique de signature** dans le README (modèle plus bas).
3. Créer un compte sur [signpath.org](https://signpath.org/) et candidater via [signpath.org/apply](https://signpath.org/apply).
4. Une fois accepté, configurer le projet dans SignPath.io et connecter le dépôt GitHub.
5. Ajouter l'action SignPath au workflow de release.
6. Chaque release est approuvée manuellement dans SignPath avant signature.

SignPath signe avec **son** certificat au nom de la fondation. Ce n'est pas un certificat personnel : c'est la fondation qui se porte garante du lien entre le dépôt et le binaire.

## Windows avec un certificat OV classique

1. Acheter un certificat de signature de code OV auprès d'une autorité reconnue (DigiCert, Sectigo, GlobalSign, SSL.com, Certum…).
2. Suivre la validation d'identité de l'autorité.
3. Récupérer le certificat au format PFX/P12 et son mot de passe.
4. Ne jamais committer le PFX ni le mot de passe.
5. Ajouter les secrets GitHub décrits plus bas, ou signer localement avec `scripts/sign-windows.ps1`.

L'autorité fournit souvent un token USB ou un HSM cloud. Pour la CI, utiliser l'outil cloud du fournisseur ou un runner qui peut accéder au HSM. Un simple fichier PFX copié dans GitHub Actions est accepté par `signtool`, mais reste un secret sensible : il doit être stocké uniquement dans les secrets du dépôt.

## macOS avec Apple Developer

1. S'inscrire à l'[Apple Developer Program](https://developer.apple.com/programs/) (99 $/an).
2. Créer dans le portail développeur :
   - un certificat **Developer ID Application** pour signer le binaire ;
   - un certificat **Developer ID Installer** pour signer le `.pkg`.
3. Exporter les deux certificats en `.p12` avec un mot de passe.
4. Créer un **mot de passe d'application** Apple pour `notarytool`.
5. Renseigner les secrets GitHub, ou exporter les variables d'environnement localement.
6. Construire le paquet avec `scripts/sign-macos.sh`, puis vérifier avec `spctl`.

Exemple de variables locales :

```bash
export APPLE_SIGN_IDENTITY="Developer ID Application: Stéphane Bamby Bazebibouta (TEAMID)"
export APPLE_INSTALLER_IDENTITY="Developer ID Installer: Stéphane Bamby Bazebibouta (TEAMID)"
export APPLE_ID="stephanebazebibouta@gmail.com"
export APPLE_TEAM_ID="TEAMID"
export APPLE_APP_PASSWORD="xxxx-xxxx-xxxx-xxxx"
./scripts/sign-macos.sh bin/braise-darwin-arm64 0.1.1
```

## Secrets GitHub

| Secret | Contenu |
| --- | --- |
| `WINDOWS_CERT_PFX_BASE64` | PFX encodé en base64 (`base64 -w0 cert.pfx`) |
| `WINDOWS_CERT_PASSWORD` | Mot de passe du PFX |
| `APPLE_CERT_P12_BASE64` | P12 Developer ID Application encodé en base64 |
| `APPLE_CERT_PASSWORD` | Mot de passe du P12 |
| `APPLE_SIGN_IDENTITY` | `Developer ID Application: Nom (TEAMID)` |
| `APPLE_INSTALLER_IDENTITY` | `Developer ID Installer: Nom (TEAMID)` |
| `APPLE_ID` | Identifiant Apple utilisé pour la notarisation |
| `APPLE_TEAM_ID` | Team ID Apple (10 caractères) |
| `APPLE_APP_PASSWORD` | Mot de passe d'application Apple |
| `GPG_PRIVATE_KEY` | Clé privée GPG, optionnelle |
| `GPG_PASSPHRASE` | Phrase de passe GPG, optionnelle |

Pour encoder un fichier :

```bash
base64 -w0 cert.pfx > cert.pfx.b64
```

Sous macOS, `base64 -i cert.p12 -o cert.p12.b64`.

Ne jamais coller ces valeurs dans une issue, un commit, un fichier de configuration ou un message public.

## Signature locale

### Windows

```powershell
pwsh ./scripts/sign-windows.ps1 `
  -Path bin/braise-windows-amd64.exe `
  -PfxPath C:\secrets\braise.pfx
```

Le script appelle `signtool`, ajoute un horodatage et vérifie la signature.

### macOS

```bash
./scripts/sign-macos.sh bin/braise-darwin-arm64 0.1.1
```

Le script :

1. importe le certificat dans un trousseau temporaire ;
2. signe le binaire avec le hardened runtime ;
3. construit un `.pkg` dans `dist/` ;
4. le signe avec le certificat Installer ;
5. le notarise avec `notarytool` ;
6. agrafe le ticket avec `stapler` ;
7. vérifie le paquet avec `spctl`.

### Empreintes

```bash
./scripts/checksums.sh bin
```

Le script produit `bin/checksums.txt`. Si `GPG_KEY_ID` est défini, il produit aussi `bin/checksums.txt.asc`.

## Vérification côté utilisateur

### Windows

```powershell
Get-FileHash .\braise-windows-amd64.exe -Algorithm SHA256
Get-AuthenticodeSignature .\braise-windows-amd64.exe | Format-List
```

`Get-AuthenticodeSignature` doit afficher `Status : Valid` et le nom de l'éditeur. Comparer l'empreinte avec `checksums.txt`.

### macOS

```bash
shasum -a 256 braise-darwin-arm64
codesign --verify --strict --verbose=2 braise-darwin-arm64
spctl -a -vv -t execute braise-darwin-arm64
```

Pour un `.pkg` :

```bash
spctl -a -vv -t install Braise-0.1.1.pkg
xcrun stapler validate Braise-0.1.1.pkg
```

### Linux

```bash
sha256sum -c checksums.txt
gpg --verify checksums.txt.asc checksums.txt
```

### Sigstore/cosign

cosign permet de vérifier qu'un fichier a été produit par un workflow GitHub précis, sans certificat d'autorité. Exemple une fois le workflow en place :

```bash
cosign verify-blob \
  --bundle checksums.txt.bundle \
  --certificate-identity-regexp 'https://github.com/foudroyant/braise/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

Cette vérification est complémentaire de la signature Windows/macOS. Elle ne supprime pas SmartScreen ni Gatekeeper.

## Ce que la signature ne garantit pas

- Elle ne garantit pas l'absence de bug ou de faille.
- Elle ne garantit pas que le logiciel fait exactement ce que vous attendez.
- Elle ne supprime pas toujours SmartScreen : la réputation se construit dans le temps.
- Elle ne remplace pas la vérification du code source ni des empreintes.
- Elle ne protège pas si l'ordinateur de l'utilisateur est déjà compromis.

## Aider les utilisateurs en attendant

En attendant une release signée :

- publier `checksums.txt` et les commandes de vérification ;
- expliquer clairement pourquoi l'avertissement apparaît ;
- proposer la compilation depuis la source ;
- ne jamais conseiller de désactiver SmartScreen ou Gatekeeper globalement ;
- indiquer les contournements ponctuels :
  - Windows : **Informations complémentaires → Exécuter quand même** ;
  - macOS : clic droit → **Ouvrir**, ou **Réglages → Confidentialité et sécurité → Ouvrir quand même**.

## Politique de signature (modèle SignPath)

À publier après acceptation du projet par SignPath Foundation, et après ajout d'une licence OSI.

```markdown
## Politique de signature

Free code signing provided by [SignPath.io](https://about.signpath.io), certificate by [SignPath Foundation](https://signpath.org).

- Committers and reviewers: [@foudroyant](https://github.com/foudroyant)
- Approvers: [@foudroyant](https://github.com/foudroyant)

Braise ne transmet aucune donnée à d'autres systèmes réseau, sauf demande explicite de l'utilisateur ou de la personne qui l'installe ou l'exploite. Le programme ne contient ni télémétrie ni appel sortant.
```

## Confidentialité

Braise fonctionne sur le réseau local. Il n'envoie pas de télémétrie et ne contacte pas de service externe. Les fichiers et messages restent dans le dossier partagé choisi par l'utilisateur. Les seules connexions sortantes sont celles que l'utilisateur déclenche lui-même (téléchargement d'un binaire, installation de polices sur le site public, par exemple).
