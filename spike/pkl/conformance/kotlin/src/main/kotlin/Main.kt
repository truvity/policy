import com.fasterxml.jackson.databind.JsonNode
import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import com.networknt.schema.JsonSchemaFactory
import com.networknt.schema.SpecVersion
import java.io.File
import org.pkl.config.kotlin.forKotlin
import org.pkl.config.kotlin.to
import org.pkl.config.java.ConfigEvaluator
import org.pkl.core.ModuleSource
import spike.contract.Showcase
import spike.contract.Service
import spike.contract.urlshortener.*

private val yaml = ObjectMapper(YAMLFactory())
private val json = ObjectMapper()
private const val BASE = "https://github.com/truvity/policy/schemas/"

private fun line(id: String, validator: String, accept: Boolean, detail: String = "") =
    println(json.writeValueAsString(mapOf("id" to id, "validator" to validator, "accept" to accept, "detail" to detail.lines().first().take(160))))

/** networknt on a schema document, resolving the shared documents from [dir] the way the repository's loader resolves them from its jar. */
private fun networknt(dir: String, root: String, instance: JsonNode): Pair<Boolean, String> {
    val factory = JsonSchemaFactory.getInstance(SpecVersion.VersionFlag.V202012) { b ->
        b.schemaMappers { m -> m.mapPrefix(BASE, "file://$dir/") }
    }
    val schema = factory.getSchema(File(root).readText())
    val errs = schema.validate(instance)
    return Pair(errs.isEmpty(), errs.minByOrNull { it.instanceLocation.toString() }?.message ?: "")
}

fun main(args: Array<String>) {
    val (spike, repo, genDir) = args
    val manifest = json.readTree(File("$spike/conformance/manifest.json"))
    val docs = json.readTree(File("$spike/conformance/documents.json"))
    ConfigEvaluator.preconfigured().forKotlin().use { ev ->
        for (e in manifest) {
            val id = e["id"].asText(); val schema = e["schema"].asText(); val path = e["path"].asText()
            val d = docs[schema]
            val instance = yaml.readTree(File(path))
            val inst = instance ?: json.nullNode()
            if (!d["hand"].isNull) {
                val r = runCatching { networknt("$repo/schemas", "$repo/${d["hand"].asText()}", inst) }
                line(id, "kt-hand", r.getOrNull()?.first ?: false, r.exceptionOrNull()?.toString() ?: r.getOrNull()!!.second)
            }
            val g = runCatching { networknt(genDir, "$genDir/${d["gen"].asText()}", inst) }
            line(id, "kt-gen", g.getOrNull()?.first ?: false, g.exceptionOrNull()?.toString() ?: g.getOrNull()!!.second)

            // The generated class alone, decoded by Jackson with no Pkl involved: types and nothing else.
            if (schema != "platform") {
                val k = runCatching {
                    val m = ObjectMapper().registerKotlinModule()
                    val text = json.writeValueAsString(inst)
                    when (schema) {
                        "web" -> m.readValue(text, Web::class.java)
                        "urls" -> m.readValue(text, Urls::class.java)
                        "redirect" -> m.readValue(text, Redirect::class.java)
                        "stat" -> m.readValue(text, Stat::class.java)
                        "prober" -> m.readValue(text, Prober::class.java)
                        "migrate" -> m.readValue(text, Migrate::class.java)
                        "log" -> m.readValue(text, LogArchiver::class.java)
                        "shortener" -> m.readValue(text, Shortener::class.java)
                        else -> m.readValue(text, Showcase::class.java)
                    }
                }
                line(id, "kt-struct", k.isSuccess, k.exceptionOrNull()?.message ?: "")
            }
            // pkl-codegen-kotlin cannot generate `platform` (it has union types): no verdict.
            if (schema == "platform") continue
            val src = """
                import "pkl:yaml"
                import "file://$spike/gen/Load.pkl"
                import "file://$spike/${d["pkl"].asText()}" as M
                local data = new yaml.Parser { useMapping = false }.parse(read("file://$path"))
                output { value = Load.load(M, data) }
            """.trimIndent()
            val r = runCatching {
                val cfg = ev.evaluateOutputValue(ModuleSource.text(src))
                when (schema) {
                    "web" -> cfg.to<Web>()
                    "urls" -> cfg.to<Urls>()
                    "redirect" -> cfg.to<Redirect>()
                    "stat" -> cfg.to<Stat>()
                    "prober" -> cfg.to<Prober>()
                    "migrate" -> cfg.to<Migrate>()
                    "log" -> cfg.to<LogArchiver>()
                    "shortener" -> cfg.to<Shortener>()
                    "showcase" -> cfg.to<Showcase>()
                    else -> error("no generated class for $schema")
                }
            }
            line(id, "pkl-kotlin", r.isSuccess, r.exceptionOrNull()?.message ?: "")
        }
    }
}
