# **teapot** é um servidor de logging
  A ideia central é ter um servidor simples que receba logs em formato plain-text e os guarde em um arquivo.
  Cada backend (ou cliente) terá uma chave de autenticação que deve ser enviada no header Authorization.
  Através dessa chave de autenticação o teapot irá identificar o backend e salvará o log em um arquivo com o exato nome do backend.
  
  Esse repositório é apenas o backend =p



## Sobre o framework/cli mug (home-brewed)
- Os arquivos dentro de ./cup são gerados automaticamente.
- Rode `mug gen` para gerar os arquivos.
- As diretivas `// mug:handler` serve para declarar um handler na rota especificada no padrão do chi.
- As variáveis de ambiente também são injetadas em `./cup/envs.go` `=)`
