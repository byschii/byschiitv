<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" indent="yes"/>
  <xsl:template match="/">
    <html>
      <head>
        <title>RTMP Status</title>
        <style>
          body { font-family: Arial, Helvetica, sans-serif; padding: 1rem; }
          table { border-collapse: collapse; width: 100%; max-width: 1000px; }
          th, td { border: 1px solid #ddd; padding: 8px; }
          th { background: #f4f4f4; text-align: left; }
        </style>
      </head>
      <body>
        <h1>nginx-rtmp Status</h1>
        <xsl:apply-templates/>
      </body>
    </html>
  </xsl:template>

  <xsl:template match="rtmp">
    <h2>Server</h2>
    <table>
      <tr><th>Key</th><th>Value</th></tr>
      <xsl:for-each select="server/*">
        <tr>
          <td><xsl:value-of select="name()"/></td>
          <td><xsl:value-of select="."/></td>
        </tr>
      </xsl:for-each>
    </table>

    <xsl:for-each select="server/application">
      <h3>Application: <xsl:value-of select="name"/></h3>
      <table>
        <tr><th>Streams</th><th>Details</th></tr>
        <xsl:for-each select="live/stream">
          <tr>
            <td><xsl:value-of select="name"/></td>
            <td>
              <xsl:value-of select="client | bw_in | bw_out | time"/>
            </td>
          </tr>
        </xsl:for-each>
      </table>
    </xsl:for-each>
  </xsl:template>
</xsl:stylesheet>
