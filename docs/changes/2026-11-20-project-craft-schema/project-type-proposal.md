# CraftSky project type and subtype proposal

This proposal gives each craft a short list of things people might look for when choosing a project to make. It draws on the current Lexicons, the 11 beta discussions and examples from craft project libraries, designers and guilds. Established project libraries show which distinctions are useful for browsing. Beta replies helped spot missing items and unclear names. [CraftSky feedback board](https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/3mpr5izppvt2k) · [Ravelry project categories](https://pressable.ravelry.com/2015/01/15/thursday-tip-pattern-less-project-categories/)

## How the picker would work

- **Type and subtype are optional.** A type groups related projects, such as Home goods. A subtype names the item, such as Cushion cover. Short lists can use item types directly, as Punch Needle does today.
- **Every type picker offers Other last.** Leave the type blank when the author skips it. Select Other to enter an optional custom item name. Named types offer their listed subtypes. Older custom names saved under named types remain visible and searchable.
- **Use the existing open subtype string.** Save a catalogue ID for a listed choice and the author’s wording for a custom entry. Trim the text and apply the existing subtype length limits. Display and keyword-search custom text. Keep standard browse filters and suggested feeds based on catalogue choices. A saved keyword or hashtag feed can cover more specific interests. [AT Protocol Lexicon string type](https://atproto.com/specs/lexicon#string) · [CraftSky search contract](../../development/search-fields.md#keyword-fields)
- **Reuse identities and aliases.** Jumper shows Pullover as a searchable alternative. Repair shows Mending the same way. Combined labels such as Coat or jacket describe distinct items that share a browse choice. A specific item can appear in a broader search. A beanie can match “hat” and a tote can match “bag”. Existing records keep their values, and old filters can continue to resolve those values when a picker label is consolidated.

The short flat lists use project types directly, with optional custom names under Other. Knitting methods, embroidery styles, crochet methods, Needle and Wet felting, and Loom type stay independent of the item choices.

## Knitting

Needle, machine and loom knitting all share these item choices. A machine-knit cardigan and a hand-knit cardigan both belong in a cardigan browse.

| Project type | Suggested subtypes |
| --- | --- |
| Garment | Jumper, Cardigan, Top, Vest or slipover, Coat or jacket, Shrug or bolero, Poncho or cape, Dress, Skirt, Baby onesie |
| Accessory | Hat, Hood or balaclava, Headband, Scarf, Cowl, Shawl or wrap, Gloves or mittens, Socks, Slippers, Bag |
| Home goods | Blanket, Baby blanket, Cushion or cover, Rug, Washcloth or dishcloth, Tea towel, Coaster or placemat, Cosy, Ornament |
| Toy or hobby | Soft toy or doll, Doll clothes, Model food |
| Pet | Pet clothing, Pet bedding, Pet toy, Pet accessory |
| Adaptive and medical | Prosthesis, Adaptive clothing |
| Parts and samples | Swatch, Motif or square, Trim or cord |
| Costume | No listed subtypes |

- **Combine labels that send people to essentially the same browse.** Beanie and toque help find Hat. Afghan and throw help find Blanket. Tee and tank help find Top. Pillow and cushion are alternate names. A single Cushion or cover choice gives both versions a shared browse. Keep Cardigan as a distinct pick. Sweater can still find cardigans and pullovers in general search. Reuse the existing Pullover identity for the closed-front subtype and preserve the older broad Sweater identity for existing projects.
- **Keep Baby blanket, Socks and Slippers independently discoverable.** Yarnspirations explicitly browses baby blankets. DROPS exposes Socks and Slippers separately under a common group. An older Craft Yarn Council survey also found hats, scarves and baby blankets among popular projects. [Yarnspirations](https://www.yarnspirations.com/pages/free-patterns) · [DROPS pattern catalogue](https://www.garnstudio.com/search.php?lang=us) · [Craft Yarn Council survey](https://craftyarncouncil.com/know.html)
- **Keep space for uncommon makes.** Beta feedback included cushion covers, lampshades, clocks, poufs and prostheses. Covers fit the shared cushion choice. Lampshades, clocks and poufs can use Home goods with the name in the description or Other with a custom name. Prosthesis gets a suggested option because there is an organised maker community with standalone patterns for these projects. [Knitting feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxrdz63v2s) · [Knitted Knockers project and patterns](https://www.knittedknockers.org/how-to-get-involved/)

## Crochet

Standard and Tunisian crochet use the same item choices. A Tunisian mosaic wall hanging is Home goods → Wall hanging, with Tunisian as the separate method and mosaic in a tag or description.

| Project type | Suggested subtypes |
| --- | --- |
| Garment | Jumper, Cardigan, Top, Vest or slipover, Coat or jacket, Poncho or cape, Dress, Skirt, Trousers or shorts, Swimwear, Baby onesie |
| Accessory | Hat, Hood or balaclava, Headband, Scarf, Cowl, Shawl or wrap, Gloves or mittens, Socks, Slippers, Bag, Bookmark |
| Home goods | Blanket, Baby blanket, Cushion or cover, Rug, Basket, Coaster or placemat, Potholder or trivet, Doily, Plant hanger, Wall hanging, Decorative plant or flower, Washcloth or dishcloth, Ornament |
| Toy or hobby | Amigurumi or soft toy, Doll clothes, Model food |
| Pet | Pet clothing, Pet bedding, Pet toy, Pet accessory |
| Adaptive and medical | Prosthesis, Adaptive clothing |
| Parts and samples | Swatch, Granny square or motif, Trim or cord |
| Costume | No listed subtypes |

- **Add the useful missing items.** Coat or jacket, Trousers or shorts, Bookmark, Wall hanging and Decorative plant or flower are defensible browse choices. Coat, hood, cushion cover and shorts also came up in beta. DMC carries crochet basket, doily and cushion cover patterns, Yarnspirations groups amigurumi and toys, and DROPS includes garments, bookmarks and decorative flowers. [Crochet feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxlzoiba2f) · [DMC crochet catalogue](https://www.dmc.com/CA/en/patterns/free-patterns-by-craft/crochet) · [Yarnspirations amigurumi and toy catalogue](https://www.yarnspirations.com/collections/patterns?project_type=Amigurumi+%26+Toys&skill_level=Beginner&skill_type=Crochet) · [DROPS pattern catalogue](https://www.garnstudio.com/search.php?lang=us)
- **Keep the item distinct from its motif.** A mandala hung on a wall belongs under Wall hanging. A mandala used as a table centrepiece can be a doily or described in the project description. Avoid making mandala a synonym for Wall hanging. A granny square belongs under Parts and samples if the square is the project. A finished granny-square bag is Accessory → Bag. A shelf-display succulent can be Decorative plant or flower. [Designer’s mandala example](https://www.ravelry.com/patterns/library/simple-flower-mandala) · [Craft Yarn Council on granny squares](https://media.craftyarncouncil.com/sites/default/files/pdfs/CIP/CYC_CIP_Crochet_Program_2017.pdf) · [DMC succulent pattern](https://www.dmc.com/US/en/products/succulent-amigurumi)
- **Group purse, tote and pouch for the first picker.** Retain those words in search. Model food covers toys, props and display pieces. Group older Soft toy and Doll choices with Amigurumi or soft toy, while preserving their saved values. Use the same alias handling for hats, neckwear and blankets as Knitting. Prosthesis also covers a documented crochet practice. [Craft Yarn Council crochet purse project](https://www.craftyarncouncil.com/node/522) · [Knitted Knockers crochet pattern](https://www.knittedknockers.org/wp-content/uploads/dlm_uploads/2023/12/KK-Crochet-Pattern-6-10-23-rev8.pdf)

## Sewing

| Project type | Suggested subtypes |
|---|---|
| Garment | Dress · Top or T-shirt · Shirt or blouse · Sweatshirt or hoodie · Cardigan · Coat or jacket · Waistcoat or gilet · Trousers · Shorts · Leggings · Skirt · Jumpsuit or romper · Dungarees · Baby bodysuit · Sleepwear or robe · Underwear or lingerie · Swimwear · Apron |
| Accessory | Bag · Tote bag · Backpack · Crossbody or sling bag · Pouch or case · Wallet or coin purse · Belt · Hat or cap · Scarf or wrap |
| Home goods | Cushion or cover · Bedding · Blanket · Curtains or blinds · Table linen · Towel or tea towel · Potholder or oven glove · Storage or organiser · Equipment cover · Rug or mat |
| Toy or hobby | Soft toy · Doll clothes · Play mat |
| Pet | Pet clothing · Pet bed · Pet toy |
| Adaptive and medical | Adaptive clothing · Wheelchair accessory · Heat or cold pack |
| Alteration and repair | Fit adjustment · Repair · Refashion |
| Costume | No listed subtypes |
| Parts and samples | Patch · Strap · Sample or toile |

Keep separate browse choices for dungarees and jumpsuits. Use Waistcoat or gilet for sleeveless outerwear. Keep “vest” searchable because it can also mean a sleeveless top. In British English, use **Trousers** with **Pants** as a search alias. Avoid making “purse” an automatic one-to-one alias since it can mean a bag or a coin purse. Clothing style, audience, and purpose can be combined with an item in search, such as `#Activewear` with Leggings or `#Formalwear` with Dress. Pattern difficulty belongs with the pattern. [The Fold Line clothing filters](https://thefoldline.com/collections/simplicity) · [Simplicity garment catalogue](https://simplicity.com/women-patterns/) · [Sewing feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxtpiww524)

Tote, backpack, and crossbody are a good first cut from the bag feedback. Keep the generic Bag option for makers who want it, and have the broad Bag browse match the named forms as well. Use **Alteration and repair** when that work is the subject of the post. A new dress with a full-bust adjustment is still a **Garment → Dress**, with the adjustment available in the description or tag. Rename “Component” to **Parts and samples** and give it a concrete prompt such as “a patch, strap, or sample you’re sharing on its own”. [The Fold Line bag catalogue](https://thefoldline.com/collections/accessories-bags) · [Sewing feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxtpiww524)

## Quilting

| Project type | Suggested subtypes |
|---|---|
| Quilt | Bed quilt · Baby or cot quilt · Throw or lap quilt · Wall quilt or hanging · Miniature quilt · Art quilt |
| Home goods | Cushion or cover · Table runner or topper · Placemat · Potholder or oven glove · Bowl cosy · Coaster · Rug or mat |
| Accessory | Bag or tote · Pouch or case |
| Garment | Jacket or coat · Waistcoat or gilet |
| Toy or hobby | Soft toy · Play mat |
| Pet | Pet bed · Pet coat |
| Parts and samples | Quilt block · Quilt top |

Keep useful intended uses such as bed, baby, and throw. Give single/twin, queen, king, and exact measurements a place in searchable descriptions. **Miniature quilt** is an established specialty with its own guild group. **Art quilt** is an established exhibition and community category. A quilt made primarily as a work of art can use Art quilt, while makers browsing for a quilt to hang on a wall can use Wall quilt or hanging. Sampler, medallion, wholecloth, patchwork, quilt-as-you-go, and memory quilt can be described and tagged alongside the item. Use **Quilt top** when the top itself is the project being shared. An in-progress bed quilt can stay **Bed quilt** and use its project status. [Missouri Star project filters](https://www.missouriquiltco.com/collections/quilt-patterns/size-throw-lap?page_view=1) · [Quilters’ Guild Miniature Group](https://www.quiltersguild.org.uk/blog/specialist-group-focus-miniature-group) · [Festival of Quilts categories](https://www.thefestivalofquilts.co.uk/wp-content/uploads/2025/08/Winners-List-2025-1.pdf) · [Quilting feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxt3sg2z27)

## Embroidery

| Project type | Suggested subtypes |
|---|---|
| Decorative piece | Hooped or framed picture · Wall hanging or banner · Sampler · Ornament · Freestanding piece |
| Home goods | Cushion or cover · Table linen · Towel or tea towel · Pincushion |
| Garment | Top or shirt · Jacket or coat · Dress or skirt |
| Accessory | Patch or badge · Bag or pouch · Brooch or jewellery · Bookmark · Needle case or scissor keep |
| Toy or hobby | Soft toy · Puppet |
| Alteration and repair | Repair |

Add item types independently from Hand/Machine and Cross-stitch/Needlepoint. A cross-stitch sampler, needlepoint cushion, and machine-embroidered jacket should each be easy to find by its finished item. Use **Sampler** for a piece made as a sampler regardless of whether it is later framed. Use **Garment → Jacket or coat** for embellishment on a jacket, and **Accessory → Patch or badge** for a standalone patch. Keep a repair type for posts where visible mending is the point. Embroidery libraries support the smaller functional items too, including pouches, bookmarks, and toys. [V&A samplers and embroidered goods](https://www.vam.ac.uk/collections/embroidery) · [Royal School of Needlework pincushion](https://royal-needlework.org.uk/product/new-helen-richman-embroidery-kits/) · [Embroidery Library project examples](https://emblibrary.com/blogs/projects) · [Embroidery feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxq53evx2m)

## Punch Needle

Keep one short picker. The existing four types are all concrete things people look for. Oxford also publishes patterns for coasters, chair pads and tabletop pieces. [Oxford pattern library](https://amyoxford.com/collections/patterns) · [Oxford project examples](https://amyoxford.com/collections/oxford-punch-needles)

**Types** Rug or floor mat, Wall hanging, Hoop art, Cushion or seat pad, Coaster, Table mat or runner.

Treat “pillow” as a search alias for Cushion or seat pad. Use Wall hanging for a piece made to hang on a wall, and Hoop art when the hoop remains its finished display frame. Reuse the existing four identifiers when changing their display labels. Patches, tote bags, and other less common outputs fit through Other and keyword search.

## Macramé

Keep one short picker here too. Plant hangers and wall hangings remain easy to find. Pattern libraries also cover bag, keyring, jewellery and tabletop projects. [DMC macramé guide](https://www.dmc.com/GB/en-GB/explore-macrame) · [DMC wall-hanging and plant-holder examples](https://www.dmc.com/PT/en-GB/learn-macrame-techniques-tutorials) · [Macramé Makers Club project library](https://macramemakers.club/macrame-tutorials-club/)

**Types** Plant hanger, Wall hanging, Bag, Keyring or charm, Jewellery, Table décor.

“Plant holder” and “keychain” are useful search aliases. Individual knots, cords and named designs belong in text or hashtags. Nålbinding is a separate way of making textiles and the feedback about it should be considered with craft coverage.

## Weaving

Add item browsing alongside the existing Loom type selector. This is the clearest case among the smaller crafts for structured item discovery. *Handwoven* offers separate Projects and Loom Type filters, with scarves/shawls, table linens, and towels among its visible project categories. Yardage can be the intended finished output of a weaving project even if someone later sews it into a garment. [Handwoven library](https://library.longthreadmedia.com/library?brand=hw) · [Handwoven on weaving yardage for clothing](https://handwovenmagazine.com/the-thrills-and-agony-of-collaboration/) · [Little Looms inkle projects](https://littlelooms.com/a-free-guide-to-inkle-weaving/)

| Project type | Suggested subtypes |
|---|---|
| Garment | Top, Jacket or coat |
| Accessory | Bag, Scarf, Shawl or wrap |
| Home goods | Towel, Table linen, Rug, Blanket or throw, Cushion cover |
| Art and decoration | Wall hanging |
| Fabric and components | Fabric yardage, Band or trim, Swatch or sampler |

Use table runner, placemat and napkin as search terms under Table linen, and use the project description to name the exact piece. Use “tapestry” as a keyword people can search across weaving projects. A maker can use tapestry techniques for other items too.

## Felting

Keep Needle and Wet as the separate method choices agreed for Felting. The International Feltmakers Association teaches combined methods and shows distinct object families including pictures, sculptures, vessels, handbags and wearables. This supports useful combinations like Wet felting + Bag and Needle felting + Sculpture or figurine. [IFA exhibition](https://www.feltmakers.com/region-8-exhibition-blog/) · [IFA feltmaking curriculum](https://www.feltmakers.com/wp-content/uploads/2016/06/IFA-CiFT-Foundation-v2.1.pdf)

| Project type | Suggested subtypes |
|---|---|
| Garment | Top, Jacket or coat, Dress |
| Accessory | Bag, Hat, Scarf or wrap, Slippers |
| Home goods | Bowl or vessel, Cushion |
| Art and decoration | Wall hanging, Sculpture or figurine, Ornament |
| Toy or hobby | Soft toy |

The existing Figurine should remain searchable with Sculpture or figurine if reclassified. A soft toy is an object intended as a toy, while Sculpture or figurine covers decorative models. Beta suggestions for repairs and embellishments are real uses of felting. Tags such as `#VisibleMending` can express repair, while a decorated garment or bag can keep its object type. [Felting beta feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxqtdtt524)

## Spinning

**Use the Spinning craft, with titles, descriptions and hashtags.** Most finished outputs in a Spinning feed are handspun yarn, so a type called Yarn adds little. Manitoba Fibre Festival distinguishes singles, plied and art yarn for judging. Art yarn can also be made by plying, so these choices express intersecting yarn characteristics. `#SinglesYarn`, `#PliedYarn` and `#ArtYarn` let those interests develop without obliging authors to choose one. Ravelry likewise connects finished handspun to yarn stash and to the projects subsequently made from it. [Manitoba Fibre Festival categories](https://www.manitobafibrefestival.com/handspun-skein-competition) · [Spin Off art yarn topics](https://spinoffmagazine.com/category/art%20yarn/1/) · [Ravelry handspun projects](https://pressable.ravelry.com/2016/07/14/thursday-tip-spinning-on-ravelry/) · [Spinning beta feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxu6zsa624)

## Dyeing

**Add one optional picker labelled “What did you dye?”** Dyeing yarn, fabric and a finished garment are different inspiration searches. The old material-form choices skewed heavily towards fibre and yarn, which the beta comments noticed. Industry sources and hobby tutorials make similar distinctions. Four choices cover the main starting points. [Cotton Council on yarn and piece dyeing](https://www.cotton.org/pubs/cottoncounts/fieldtofabric/dyeing.cfm) · [Ashford dyeing tutorials](https://www.ashford.co.nz/dyeing-tutorials/) · [Dharma dyeing tutorials](https://www.dharmatrading.com/pages/how-to-instructions) · [Dyeing beta feedback](https://userinput.app/d/did:plc:lmmx63zcns6gewgxqfdt4kof/3n2sxoj3knx2a)

**Types** Unspun fibre, Yarn or thread, Fabric, Finished item.

“Finished item” covers an already assembled garment, accessory or household textile. If a project includes several forms, the main example can be the selected item and the details added to the description. Dye source, colour effect and application, such as natural dyeing and ice dyeing, are independent topics for tags or descriptions.
