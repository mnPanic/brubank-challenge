# brubank-challenge

Challenge de entrevista de Brubank, Febrero 2023

TODOs:

- Refactor calculo de llamadas para que quede lindo y sea extensible.
- Validar formato números de teléfono
- Llamada a si mismo
- Mover test data de usuarios y teléfonos para evitar repetición
- Interfaz de servicio web

Notas sobre el servicio de consulta de usuarios

https://interview-brubank-api.herokuapp.com/users/:phoneNumber

La respuesta es un JSON con el siguiente formato

```json
{
    "address": "Address of the user",
    "name": "Name of the user",
    "phone_number": "+xxxxxxxxx",
    "friends": ["+xxxxxxxxx","+xxxxxxxxx"],
}
```

Dudas:

- Qué pasa si se llama a si mismo?
- Puede tener amigos internacionales o solo nacionales?
  - Según el ejemplo, una llamada con un amigo nacional se cuenta en
    total_national_seconds y total_friends_seconds. Como se dice que hay 3 tipos
    de llamadas, esto me hace pensar que toda llamada de amigos es nacional
    (sino debería haber más tipos de llamadas, las combinaciones amigo-nacional,
    amigo-internacional, extraño-nacional, extraño-internacional)
- Qué hacer con las llamadas que tengan un número origen que no sea el del
  usuario para generar factura. Error? Ignorar?
- Qué debería pasar con las llamadas que están fuera del período de facturación?
- Fecha de llamadas del CSV en UTC, qué pasa si no está en ese timezone? Se
  convierte? Se devuelve un error? Lo tengo que validar seguro
- README en castellano seguro. Código en inglés. Pero comentarios, prefieren en
  castellano o inglés?

Notas de decisiones y separación de responsabilidades

- El hecho de que las llamadas vengan en un CSV es accidental, bien podría ser
  un array en un JSON. Por esa razón la lógica de parseo queda del lado del
  handler.
- Para los assertions usé testify, que es una gran lib.
- Decidí que en caso de que haya un error de formato en un entry, que se frene
  todo el proceso especificando el error y solicitando que se corrija. Esto me
  parece mejor que informarlo de una forma que no frene el proceso y dar una
  factura incompleta, ya que después esos errores se terminan ignorando y se
  sigue con el proceso a menos que tenga *hards stops*. En este negocio, eso
  terminaría con que tal vez le cobremos menos a los usuarios. Por ejemplo, si
  se carga el número de teléfono del usuario con un dígito menos, lo
  descartaríamos como inválido y no lo tendríamos en cuenta para su factura,
  haciendo que la empresa pierda plata.
- Decidí hacer un CLI en lugar de un servicio web porque resultaba más fácil
  leer el CSV.

Referencias:

- https://stackoverflow.com/questions/38596079/how-do-i-parse-an-iso-8601-timestamp-in-go

## Uso

Recomendación: instalar jq