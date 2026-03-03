# Haproxy CLI e Agent

Project Name <CLI name>: hapctl

## Context

A ideia é, criar uma CLI que também seja um Agent, que fique monitorando um diretório. Dentro desse diretório, qualquer arquivo que esteja no padrão, pode ser até dentro de sub diretórios, ele possa atualizar as configurações do HAProxy. O formato do arquivo será em yaml. Caso algum bind apresente problema, ele possa emitir um alerta, chamando um webhook, via post, tendo a opção de headers. O body será sempre padrão, contendo o nome do bind que apresentou problema.

## Rules

- Implementação com a liguagem Go, com código bem organizado e estruturado
- Código, comments, documentação tudo em Inglês
- Implementação dá lógica bem organizada, código desacoplado, em pacotes. Siguindo boas práticas de programação.

## TODO

**Examplo de uma config**
```yml
binds:
  - name: game-server
    override: true # optional, if not set, it will be false. If true, the config will override the existing config
    enabled: true # optional, if not set, it will be true. If false, the config will be removed from the HAProxy config
    description: Game server bind
    type: tcp
    ip: * # optional, if not set, it will be * (all interfaces)
    port: 7777
```
- [x] Gitignore para não armazenar o que não for preciso
- [x] Utilizar lib leita de argumentos, como cobra, tipo o que é feito em no projeto stackctl
- [x] Código core deve ser desacoplado no sistema de leitura dos argumentos e de arquivos, pois deve ser possível usar via argumentos também. Ao passar o valores, deve gerar o arquivo, seguindo os padrões
- [x] Deve ser possível informar o path onde será salvo/criado/ler os arquivos para sicronizar o HAProxy config
- [x] Deve ser possível informar o path do config do da cli, onde terá as informações de execução, tipo o arquivo de configuração.
  ```yaml
  sync:
    # Path where the files will be read
    resource-path: /path/to/resource # default /etc/haproxy/hapctl/resources
    # Interval to sync the files
    interval: 5s
    # Enable sync
    enabled: true
  monitoring:
   enabled: true # defualt true if true, base on the interval, will check the status for each declared bind
   interval: 5s # default 5s
   webhook: # if not provided, data result will be stored in logs /var/log/hapctl/monitoring.log
    url: http://localhost:8080/webhook
    headers:
      - name: Content-Type
        value: application/json
      - name: ApiKey
        value: 123456
  ```
- [x] Arquivos com extensão yaml ou yml
- [x] Todos os logs devem ser armazenados em um arquivo, com o nome hapctl.log, com o caminho /var/log/hapctl.log e expirar em 7 dias
- [x] Implementar o sistema de monitoramento, onde de tempos em tempos ele verifica se todos os binds estão funcionando e envia via webhook o status de cada bind
- [x] Implementar lógica que, ao add uma configuração válida, a mesma seja aplicada ao HAProxy confg e o haproxy seja atualizado.
- [x] Cada arquivo pode ter uma lista de configurações a seram aplicadas.
- [x] Implementar lógica que, ao add uma configuração inválida, ele possa emitir um alerta, chamando um webhook, via post, tendo a opção de headers. O body será sempre padrão, contendo o nome do bind que apresentou problema.
- [x] Unit tests para todos os pacotes
- [x] Action para validar do testes
- [x] Documentação clara, contento o objetivo e como usar, configurar