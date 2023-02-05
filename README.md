<!-- omit in toc -->
# Brubank Challenge

- [Cómo ejecutar](#cómo-ejecutar)
- [Enunciado y aclaraciones](#enunciado-y-aclaraciones)
- [Notas sobre desarrollo](#notas-sobre-desarrollo)
  - [Paquetes](#paquetes)
  - [Tests](#tests)
  - [Cálculo de llamadas extensible](#cálculo-de-llamadas-extensible)
  - [Manejo de errores](#manejo-de-errores)
  - [Dependencias externas](#dependencias-externas)

Challenge de entrevista para Brubank, Febrero 2023.

Aspirante: Manuel Panichelli (panicmanu@gmail.com /
https://www.linkedin.com/in/manuel-panichelli/)

TODOs:

- Mover test data de usuarios y teléfonos para evitar repetición

## Cómo ejecutar

La solución es un CLI (me pareció lo más sencillo de implementar por la lectura
de CSVs). Se le pasan los datos de entrada como especifica el enunciado y el
output es la factura en JSON. Como este no tiene formato, recomiendo instalar
[jq](https://stedolan.github.io/jq/) para poder visualizarlo mejor.

Lo desarrollé con la versión go1.18.4. Debería andar bien con versiones
posteriores, pero cualquier cosa pueden probar con esa.

Los argumentos son posicionales,

1. Número de teléfono del usuario a generar la factura. Debe tener un formato correcto
2. Fecha de inicio del período de facturación (`AAAA-MM-DD`)
3. Fecha de fin del período de facturación (`AAAA-MM-DD`)
4. Path al CSV con la lista de llamadas. Se espera que la primera fila sea el
   header, y que las columnas sean número destino, número origen, duración (en
   segundos), fecha (ISO8601 en UTC)

Ejemplo de uso (usando el `csv` provisto):

```bash
$ go run main.go +5491167930920 2020-01-01 2022-12-12 enunciado/example-brubank-challenge.csv | jq
Generated invoice successfully
{
  "user": {
    "address": "562 Ritchie Mall",
    "name": "Bradford Reichel",
    "phone_number": "+5491167930920"
  },
  "calls": [
    {
      "phone_number": "+5491167940999",
      "duration": 484,
      "timestamp": "2021-04-02T11:09:02Z",
      "amount": 2.5
    },
    // ...
    {
      "phone_number": "+5491167940999",
      "duration": 72,
      "timestamp": "2020-10-05T10:07:09Z",
      "amount": 2.5
    }
  ],
  "total_international_seconds": 6042,
  "total_national_seconds": 15831,
  "total_friends_seconds": 7172,
  "total": 5245.5
}
```

Correr tests:

```bash
go test ./...
```

Correr tests con coverage:

```bash
go test ./... -coverpkg=./... -coverprofile cover.out -covermode=count
go tool cover -html=cover.out
```

## Enunciado y aclaraciones

En el directorio [`enunciado/`](enunciado) está el enunciado y datos de ejemplo.
Por mail tuve las siguientes aclaraciones:

- Puede haber llamadas de usuarios que no sean el especificado y que estén fuera
  del período de facturación, hay que filtrarlas.
- Que un usuario se llame a sí mismo es válido
- Las llamadas tienen dos dimensiones de tipos independientes: amigo/extraño y
  nacional/internacional (puede haber llamadas internacionales a amigos)
- La extensibilidad de nuevas llamadas se explicó con dos ejemplos. Se debe
  poder agregar cosas como esas sin tener que modificar mucho código.
  - Nuevo tipo de llamadas interplanetarias, que afecta a los segundos totales y
    cálculo de costos.
  - Nuevo tipo de promoción para llamadas internacionales pero a países del
    Mercosur.

El servicio para consulta de usuarios tiene la siguiente URL:
`https://fn-interview-api.azurewebsites.net/users/{phoneNumber}`. No retorna
errores y la respuesta es un JSON con formato

```json
{
    "address": "Address of the user",
    "name": "Name of the user",
    "phone_number": "+xxxxxxxxx",
    "friends": ["+xxxxxxxxx","+xxxxxxxxx"],
}
```

## Notas sobre desarrollo

El código está escrito y documentado en inglés, salvo "meta" notas de diseño que
están en castellano (no estarían en un código productivo, pero son dirigidas al
corrector).

Dejé los commits mas o menos tal cual los fuí haciendo mientras desarrollaba,
por si quieren ver el proceso que tomé. En una situación de la vida real,
probablemente haría un rebase y juntaría los commits con algún criterio.

### Paquetes

Separé las responsabilidades del problema en los siguientes paquetes,

- [`main`](main.go): Entry point del programa, llama a CLI
- [`cli`](cmd/cli/cli.go): Tiene la interfaz pedida por el enunciado. Parsea el
  CSV a tipos de Go y delega el creado de la factura al paquete `invoice`.

  Interpreté que el hecho de que las llamadas vengan en un CSV es algo que tiene
  que ver con la interfaz, pero no con la lógica de negocio del armado de
  facturas. Bien podría ser un array en un JSON. Por esa razón la lógica de
  parseo queda del lado del handler.

- [`invoice`](pkg/invoice/): Dada una lista de llamadas y un número de
  teléfono, busca al usuario en el servicio (usando un `user.Finder`) itera las
  llamadas para calcular su costo (usando `call.Processor`) y devuelve una
  factura.
- [`user`](pkg/user/): Definición de usuario y `Finder`, que consume el
  servicio de Brubank. También brinda un mock sencillo (Nota: podría haber
  estado en un pkg `usermock` pero me pareció más simple en este caso que esté
  todo junto)
- [`call`](pkg/invoice/call/): Brinda un *procesador de llamadas* que calcula
  los costos y resume las duraciones totales. Separé la
  lógica de negocio de costeo de llamadas de la generación de facturas, con la
  justificación de que se podría querer costear una llamada para un contexto
  diferente.

### Tests

Los tests principales de la lógica de negocio son sobre el pkg `invoice`, dado
que es más cómodo que hacerlo desde el CLI (por la interfaz). En el CLI solo
testié lo que no hice en `invoice`, lo referido al parseo del input.

Queda como trabajo futuro refactorizar un poco los tests para evitar que en
todos se repita la definición de usuarios y llamadas de test.

### Cálculo de llamadas extensible

Esta parte fue la que más tiempo me llevó diseñar. Me resultó difícil encontrar
abstracciones que expresen correctamente que las llamadas tienen diferentes
tipos (nacional, internacional, amigo y eventualmente interplanetaria) y que el
cálculo de costos varía según el tipo y otras características adicionales.

A lo que llegué es lo siguiente: Las llamadas tienen **tipos**, que pueden ser
compuestos. Estos determinan el costo base. Luego, se puede tener
**promociones**, que se configuran en una lista y se chequea en orden si alguna
se puede aplicar y de ser así se retorna su costo. Sino, el costo base.

Esto permite modelar el problema de la siguiente forma:

- Las llamadas nacionales, internacionales e interplanetarias se modelan como
  tipos de llamada.
- Las llamadas a amigos y que las primeras 10 son gratis se divide en dos:
  - Las llamadas a amigos son un tipo de llamada compuesto (que por debajo
  conoce el tipo base, para poder registrar la duración de ambas)
  - El hecho de que las primeras 10 sean gratis se configura como una promoción.
   El contador de cuantas llamadas fueron gratis (para que la 11ava se cobre)
   queda encapsulado dentro, así no se acopla con el mecanismo de aplicado de
   promociones y procesamiento de llamadas.
- Que las llamadas internacionales al mercosur tengan descuento se modela como
  una promoción.

No se cuentan los segundos totales para las promociones, pero sí para los tipos
de llamadas (por eso las llamadas a amigos tienen que ser un tipo, sino serían
solo promo como mercosur). Para ello el tipo de llamada tiene un método
`RegisterDuration` que vuelve al procesador con `RegisterFriendCall`,
`RegisterNationalCall` o `RegisterInternationalCall` (tipo double dispatch).

Esto permite extenderlo con nuevos tipos de llamadas tocando relativamente poco
código (y al menos ese código no es parte del core de procesamiento de llamadas),

- Para agregar una nueva **promoción**, se debe crear un struct que implemente
  [`call.Promotion`](pkg/invoice/call/promotions.go). Para tenerla en cuenta en
  el procesamiento de llamadas, se agrega a la lista de promociones en el
  `call.Processor` que se crea en `invoice.Generate()`
- Para agregar un nuevo **tipo de llamada** que tenga contados los segundos
  totales,
  - Crear una estructura que implemente `call.Type`
  - Extender `Call.Type()` para que se devuelva ese tipo cuando sea necesario

    > Nota: Considero necesario tener una función que seleccione el tipo en
    > lugar de por ejemplo una lista de tipos con criterios para aplicar para
    > estar seguros de que los criterios no se superponen (ya que una llamada no
    > puede tener más de un tipo).
    >
    > Este no es el caso de las promociones, en donde diferentes promociones
    > pueden aplicar a la misma llamada. Para ellas, la precedencia está dictada
    > según el orden en el que se configuren (se usa la primera que aplica).

  - Agregar la duración a `invoice.Invoice`, `call.TotalDurations` y el método
    de registrado correspondiente a `call.DurationRegisterer` y
    `call.Processor`.

    > Esto es lo que me parece que quedó más raro, pero no se me ocurrió otra
    > forma de hacerlo. Como las duraciones totales van a terminar en
    > `Invoice` como atributos de un struct, no se puede acceder de forma
    > genérica tipo `total_{type}_seconds` (excepto con reflection, pero es
    > overkill). Tampoco me gustaría tenerla como `map` para poder hacerlo.

Dejé en el branch `example-extend` una implementación de ejemplo de las
llamadas interplanetarias y la promoción de llamadas a países del mercosur. Se
puede ver en el [PR #1](https://github.com/mnPanic/brubank-challenge/pull/1).

Nota: Si no hubiera sido porque el enunciado pide explícitamente programar una
lógica extensible para cálculo de llamadas, con las cosas que tiene por ahora lo
hubiera dejado "menos diseñado" y con ifs, y hubiera esperado a tener más casos
implementados para generar una abstracción (por lo menos interplanetarias y la
promo de Mercosur).

### Manejo de errores

Decidí que en caso de que haya un error de formato en un entry, se frene
todo el proceso especificando el error y solicitando que se corrija. Esto me
parece mejor que informarlo de una forma que no frene el proceso y dar una
factura incompleta, ya que después esos errores se terminan ignorando y se
sigue con el proceso a menos que tenga *hards stops*.

En este negocio, eso terminaría con que tal vez le cobremos menos a los
usuarios. Por ejemplo, si se carga el número de teléfono del usuario con un
dígito menos, lo descartaríamos como inválido y no lo tendríamos en cuenta para
su factura, haciendo que la empresa pierda plata.

### Dependencias externas

La única dependencia que usé es [testify](github.com/stretchr/testify) para los
assertions, porque me parece muy cómoda.
