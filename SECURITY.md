# Sécurité

## Signaler une vulnérabilité

Ouvrez un [avis de sécurité privé GitHub](https://github.com/foudroyant/braise/security/advisories/new) plutôt qu'une issue publique. Indiquez :

- la version concernée ;
- le système et le navigateur ;
- les étapes de reproduction ;
- l'impact estimé.

Ne publiez pas de détails exploitables avant qu'un correctif soit disponible.

## Modèle de menace

Braise est conçu pour un **réseau local de confiance**. Il n'y a pas d'authentification par défaut : toute personne qui atteint le port peut lire, déposer et discuter. Les mesures présentes sont décrites dans le README (chemins confinés, refus des liens symboliques, vérification d'origine, CSP, limites d'envoi, mode lecture seule).

Pour un réseau non fiable, isolez le port dans le pare-feu ou utilisez `--read-only`. N'exposez pas Braise directement sur Internet.

## Signature des binaires

Les binaires de la version 0.1.0 ne sont pas signés. Les options de signature Windows/macOS, les coûts, les secrets et la procédure de vérification sont documentés dans [`SIGNING.md`](SIGNING.md).

## Versions prises en charge

Les correctifs de sécurité sont appliqués à la dernière version publiée. Les versions antérieures ne sont pas maintenues.
