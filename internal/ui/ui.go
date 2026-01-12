package ui

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/panozzaj/roost-dev/internal/styles"
)

// dashboardData contains all data passed to the dashboard template
type dashboardData struct {
	ThemeScript template.HTML
	ThemeCSS    template.CSS
	MarkCSS     template.CSS
	TLD         string
	Port        int
	InitialData template.JS
}

// dashboardTmpl is the parsed template for the dashboard page
var dashboardTmpl = template.Must(template.New("dashboard").Parse(dashboardHTML))

// ServeIndex serves the main dashboard HTML with initial app data
func ServeIndex(w http.ResponseWriter, r *http.Request, tld string, port int, initialData []byte, theme string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := dashboardData{
		ThemeScript: template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:    template.CSS(styles.HeadCSS()),
		MarkCSS:     template.CSS(styles.LogsCSS()),
		TLD:         tld,
		Port:        port,
		InitialData: template.JS(string(initialData)),
	}

	var b strings.Builder
	dashboardTmpl.Execute(&b, data)
	w.Write([]byte(b.String()))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>roost-dev</title>
{{.ThemeScript}}
    <style>
{{.ThemeCSS}}

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            padding: 20px;
            transition: background 0.2s, color 0.2s;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
        }
        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px solid var(--border-color);
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
        }
        .header-actions {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .theme-toggle {
            background: var(--btn-bg);
            color: var(--text-secondary);
            border: none;
            padding: 8px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 16px;
            line-height: 1;
        }
        .theme-toggle:hover {
            background: var(--btn-hover);
        }
        .connection-status {
            font-size: 12px;
            color: var(--text-muted);
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .connection-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--error);
        }
        .connection-dot.connected {
            background: var(--success);
        }
        .app {
            background: var(--bg-secondary);
            border-radius: 8px;
            margin-bottom: 12px;
            transition: background 0.2s;
        }
        .app.highlight {
            animation: highlightPulse 1s ease-out;
        }
        @keyframes highlightPulse {
            0% { box-shadow: 0 0 0 0 rgba(96, 165, 250, 0.7); }
            70% { box-shadow: 0 0 0 10px rgba(96, 165, 250, 0); }
            100% { box-shadow: 0 0 0 0 rgba(96, 165, 250, 0); }
        }
        .app-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 20px;
            cursor: pointer;
        }
        .app-header:hover {
            background: var(--bg-tertiary);
        }
        .app-info {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .status-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            background: var(--text-muted);
            cursor: pointer;
            transition: transform 0.1s;
        }
        .status-dot:hover {
            transform: scale(1.2);
            box-shadow: 0 0 6px currentColor;
        }
        .status-dot-wrapper {
            position: relative;
            display: inline-block;
            padding: 8px;
            margin: -8px;
        }
        .status-menu {
            position: absolute;
            top: 100%;
            left: 50%;
            transform: translateX(-50%);
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 4px 0;
            min-width: 80px;
            z-index: 100;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            display: none;
        }
        .status-menu.visible {
            display: block;
        }
        .status-menu button {
            display: block;
            width: 100%;
            padding: 6px 12px;
            background: none;
            border: none;
            color: var(--text-secondary);
            font-size: 12px;
            text-align: left;
            cursor: pointer;
        }
        .status-menu button:hover {
            background: var(--btn-bg);
            color: var(--text-primary);
        }
        .status-menu button.danger {
            color: var(--error);
        }
        .status-menu button.danger:hover {
            background: var(--error);
            color: #fff;
        }
        .status-dot.running {
            background: var(--success);
        }
        .status-dot.failed {
            background: var(--error);
        }
        .status-dot.idle {
            background: var(--text-muted);
        }
        .status-dot.starting {
            background: var(--warning);
            animation: pulse 1s ease-in-out infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.5; transform: scale(1.2); }
        }
        .app-description {
            font-size: 13px;
            color: var(--text-muted);
            margin-left: 4px;
        }
        .app-name {
            font-weight: 600;
            font-size: 16px;
        }
        .app-aliases {
            font-size: 12px;
            color: var(--text-muted);
            font-style: italic;
        }
        .app-url {
            color: var(--accent-blue);
            text-decoration: none;
            font-size: 14px;
        }
        .app-url:hover {
            text-decoration: underline;
            color: var(--accent-blue-hover);
        }
        .app-meta {
            display: flex;
            align-items: center;
            gap: 16px;
        }
        .app-port {
            font-size: 14px;
            color: var(--text-muted);
        }
        .app-uptime {
            font-size: 13px;
            color: var(--text-muted);
            min-width: 40px;
        }
        .services {
            padding: 0 8px 16px 42px;
        }
        .service {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px 12px;
            background: var(--bg-tertiary);
            border-radius: 4px;
            margin-top: 8px;
        }
        .service-info {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .service-name {
            font-size: 14px;
            color: var(--text-secondary);
        }
        .service-meta {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .app-error {
            font-size: 12px;
            color: var(--error);
            display: block;
            margin-top: 4px;
            padding: 4px 8px;
            background: var(--error-bg);
            border-radius: 4px;
            max-width: 500px;
        }
        .logs-panel {
            background: var(--bg-logs);
            border-top: 1px solid var(--border-color);
            padding: 16px 20px;
            display: none;
        }
        .logs-panel.visible {
            display: block;
        }
        .logs-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;
        }
        .logs-title {
            font-size: 14px;
            color: var(--text-muted);
        }
        .logs-actions {
            display: flex;
            gap: 8px;
        }
        .logs-actions button {
            background: var(--btn-bg);
            color: var(--text-secondary);
            border: none;
            padding: 6px;
            border-radius: 4px;
            cursor: pointer;
            transition: background 0.15s;
            display: none;
        }
        .logs-actions button.visible {
            display: inline-flex;
        }
        .logs-actions button:hover {
            background: var(--btn-hover);
            color: var(--text-primary);
        }
        .logs-actions button svg {
            width: 16px;
            height: 16px;
        }
        .logs-actions .claude-btn {
            animation: claude-pulse 5s ease-in-out infinite;
        }
        @keyframes claude-pulse {
            0%, 40%, 100% { color: #da7756; }
            50%, 90% { color: #C15F3C; }
        }
        .logs-content {
            font-family: "SF Mono", Monaco, "Cascadia Code", monospace;
            font-size: 12px;
            line-height: 1.6;
            max-height: 300px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            color: var(--text-secondary);
        }
{{.MarkCSS}}
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--text-muted);
        }
        .empty-state h2 {
            font-size: 18px;
            margin-bottom: 12px;
            color: var(--text-secondary);
        }
        .empty-state code {
            display: block;
            background: var(--bg-secondary);
            padding: 16px;
            border-radius: 6px;
            margin-top: 16px;
            font-family: "SF Mono", Monaco, monospace;
            font-size: 13px;
            color: #7c3aed;
            text-align: left;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>roost-dev</h1>
            <div class="header-actions">
                <span class="connection-status">
                    <span class="connection-dot" id="connection-dot"></span>
                    <span id="connection-text">Connecting...</span>
                </span>
                <button class="theme-toggle" onclick="toggleTheme()" title="Toggle theme">
                    <span id="theme-icon">&#9790;</span>
                </button>
            </div>
        </header>
        <main id="apps"></main>
    </div>

    <script>
        // morphdom v2.7.4 - https://github.com/patrick-steele-idem/morphdom
        (function(global,factory){typeof exports==="object"&&typeof module!=="undefined"?module.exports=factory():typeof define==="function"&&define.amd?define(factory):(global=global||self,global.morphdom=factory())})(this,function(){"use strict";var DOCUMENT_FRAGMENT_NODE=11;function morphAttrs(fromNode,toNode){var toNodeAttrs=toNode.attributes;var attr;var attrName;var attrNamespaceURI;var attrValue;var fromValue;if(toNode.nodeType===DOCUMENT_FRAGMENT_NODE||fromNode.nodeType===DOCUMENT_FRAGMENT_NODE){return}for(var i=toNodeAttrs.length-1;i>=0;i--){attr=toNodeAttrs[i];attrName=attr.name;attrNamespaceURI=attr.namespaceURI;attrValue=attr.value;if(attrNamespaceURI){attrName=attr.localName||attrName;fromValue=fromNode.getAttributeNS(attrNamespaceURI,attrName);if(fromValue!==attrValue){if(attr.prefix==="xmlns"){attrName=attr.name}fromNode.setAttributeNS(attrNamespaceURI,attrName,attrValue)}}else{fromValue=fromNode.getAttribute(attrName);if(fromValue!==attrValue){fromNode.setAttribute(attrName,attrValue)}}}var fromNodeAttrs=fromNode.attributes;for(var d=fromNodeAttrs.length-1;d>=0;d--){attr=fromNodeAttrs[d];attrName=attr.name;attrNamespaceURI=attr.namespaceURI;if(attrNamespaceURI){attrName=attr.localName||attrName;if(!toNode.hasAttributeNS(attrNamespaceURI,attrName)){fromNode.removeAttributeNS(attrNamespaceURI,attrName)}}else{if(!toNode.hasAttribute(attrName)){fromNode.removeAttribute(attrName)}}}}var range;var NS_XHTML="http://www.w3.org/1999/xhtml";var doc=typeof document==="undefined"?undefined:document;var HAS_TEMPLATE_SUPPORT=!!doc&&"content"in doc.createElement("template");var HAS_RANGE_SUPPORT=!!doc&&doc.createRange&&"createContextualFragment"in doc.createRange();function createFragmentFromTemplate(str){var template=doc.createElement("template");template.innerHTML=str;return template.content.childNodes[0]}function createFragmentFromRange(str){if(!range){range=doc.createRange();range.selectNode(doc.body)}var fragment=range.createContextualFragment(str);return fragment.childNodes[0]}function createFragmentFromWrap(str){var fragment=doc.createElement("body");fragment.innerHTML=str;return fragment.childNodes[0]}function toElement(str){str=str.trim();if(HAS_TEMPLATE_SUPPORT){return createFragmentFromTemplate(str)}else if(HAS_RANGE_SUPPORT){return createFragmentFromRange(str)}return createFragmentFromWrap(str)}function compareNodeNames(fromEl,toEl){var fromNodeName=fromEl.nodeName;var toNodeName=toEl.nodeName;var fromCodeStart,toCodeStart;if(fromNodeName===toNodeName){return true}fromCodeStart=fromNodeName.charCodeAt(0);toCodeStart=toNodeName.charCodeAt(0);if(fromCodeStart<=90&&toCodeStart>=97){return fromNodeName===toNodeName.toUpperCase()}else if(toCodeStart<=90&&fromCodeStart>=97){return toNodeName===fromNodeName.toUpperCase()}else{return false}}function createElementNS(name,namespaceURI){return!namespaceURI||namespaceURI===NS_XHTML?doc.createElement(name):doc.createElementNS(namespaceURI,name)}function moveChildren(fromEl,toEl){var curChild=fromEl.firstChild;while(curChild){var nextChild=curChild.nextSibling;toEl.appendChild(curChild);curChild=nextChild}return toEl}function syncBooleanAttrProp(fromEl,toEl,name){if(fromEl[name]!==toEl[name]){fromEl[name]=toEl[name];if(fromEl[name]){fromEl.setAttribute(name,"")}else{fromEl.removeAttribute(name)}}}var specialElHandlers={OPTION:function(fromEl,toEl){var parentNode=fromEl.parentNode;if(parentNode){var parentName=parentNode.nodeName.toUpperCase();if(parentName==="OPTGROUP"){parentNode=parentNode.parentNode;parentName=parentNode&&parentNode.nodeName.toUpperCase()}if(parentName==="SELECT"&&!parentNode.hasAttribute("multiple")){if(fromEl.hasAttribute("selected")&&!toEl.selected){fromEl.setAttribute("selected","selected");fromEl.removeAttribute("selected")}parentNode.selectedIndex=-1}}syncBooleanAttrProp(fromEl,toEl,"selected")},INPUT:function(fromEl,toEl){syncBooleanAttrProp(fromEl,toEl,"checked");syncBooleanAttrProp(fromEl,toEl,"disabled");if(fromEl.value!==toEl.value){fromEl.value=toEl.value}if(!toEl.hasAttribute("value")){fromEl.removeAttribute("value")}},TEXTAREA:function(fromEl,toEl){var newValue=toEl.value;if(fromEl.value!==newValue){fromEl.value=newValue}var firstChild=fromEl.firstChild;if(firstChild){var oldValue=firstChild.nodeValue;if(oldValue==newValue||!newValue&&oldValue==fromEl.placeholder){return}firstChild.nodeValue=newValue}},SELECT:function(fromEl,toEl){if(!toEl.hasAttribute("multiple")){var selectedIndex=-1;var i=0;var curChild=fromEl.firstChild;var optgroup;var nodeName;while(curChild){nodeName=curChild.nodeName&&curChild.nodeName.toUpperCase();if(nodeName==="OPTGROUP"){optgroup=curChild;curChild=optgroup.firstChild}else{if(nodeName==="OPTION"){if(curChild.hasAttribute("selected")){selectedIndex=i;break}i++}curChild=curChild.nextSibling;if(!curChild&&optgroup){curChild=optgroup.nextSibling;optgroup=null}}}fromEl.selectedIndex=selectedIndex}}};var ELEMENT_NODE=1;var DOCUMENT_FRAGMENT_NODE$1=11;var TEXT_NODE=3;var COMMENT_NODE=8;function noop(){}function defaultGetNodeKey(node){if(node){return node.getAttribute&&node.getAttribute("id")||node.id}}function morphdomFactory(morphAttrs){return function morphdom(fromNode,toNode,options){if(!options){options={}}if(typeof toNode==="string"){if(fromNode.nodeName==="#document"||fromNode.nodeName==="HTML"||fromNode.nodeName==="BODY"){var toNodeHtml=toNode;toNode=doc.createElement("html");toNode.innerHTML=toNodeHtml}else{toNode=toElement(toNode)}}else if(toNode.nodeType===DOCUMENT_FRAGMENT_NODE$1){toNode=toNode.firstElementChild}var getNodeKey=options.getNodeKey||defaultGetNodeKey;var onBeforeNodeAdded=options.onBeforeNodeAdded||noop;var onNodeAdded=options.onNodeAdded||noop;var onBeforeElUpdated=options.onBeforeElUpdated||noop;var onElUpdated=options.onElUpdated||noop;var onBeforeNodeDiscarded=options.onBeforeNodeDiscarded||noop;var onNodeDiscarded=options.onNodeDiscarded||noop;var onBeforeElChildrenUpdated=options.onBeforeElChildrenUpdated||noop;var skipFromChildren=options.skipFromChildren||noop;var addChild=options.addChild||function(parent,child){return parent.appendChild(child)};var childrenOnly=options.childrenOnly===true;var fromNodesLookup=Object.create(null);var keyedRemovalList=[];function addKeyedRemoval(key){keyedRemovalList.push(key)}function walkDiscardedChildNodes(node,skipKeyedNodes){if(node.nodeType===ELEMENT_NODE){var curChild=node.firstChild;while(curChild){var key=undefined;if(skipKeyedNodes&&(key=getNodeKey(curChild))){addKeyedRemoval(key)}else{onNodeDiscarded(curChild);if(curChild.firstChild){walkDiscardedChildNodes(curChild,skipKeyedNodes)}}curChild=curChild.nextSibling}}}function removeNode(node,parentNode,skipKeyedNodes){if(onBeforeNodeDiscarded(node)===false){return}if(parentNode){parentNode.removeChild(node)}onNodeDiscarded(node);walkDiscardedChildNodes(node,skipKeyedNodes)}function indexTree(node){if(node.nodeType===ELEMENT_NODE||node.nodeType===DOCUMENT_FRAGMENT_NODE$1){var curChild=node.firstChild;while(curChild){var key=getNodeKey(curChild);if(key){fromNodesLookup[key]=curChild}indexTree(curChild);curChild=curChild.nextSibling}}}indexTree(fromNode);function handleNodeAdded(el){onNodeAdded(el);var curChild=el.firstChild;while(curChild){var nextSibling=curChild.nextSibling;var key=getNodeKey(curChild);if(key){var unmatchedFromEl=fromNodesLookup[key];if(unmatchedFromEl&&compareNodeNames(curChild,unmatchedFromEl)){curChild.parentNode.replaceChild(unmatchedFromEl,curChild);morphEl(unmatchedFromEl,curChild)}else{handleNodeAdded(curChild)}}else{handleNodeAdded(curChild)}curChild=nextSibling}}function cleanupFromEl(fromEl,curFromNodeChild,curFromNodeKey){while(curFromNodeChild){var fromNextSibling=curFromNodeChild.nextSibling;if(curFromNodeKey=getNodeKey(curFromNodeChild)){addKeyedRemoval(curFromNodeKey)}else{removeNode(curFromNodeChild,fromEl,true)}curFromNodeChild=fromNextSibling}}function morphEl(fromEl,toEl,childrenOnly){var toElKey=getNodeKey(toEl);if(toElKey){delete fromNodesLookup[toElKey]}if(!childrenOnly){var beforeUpdateResult=onBeforeElUpdated(fromEl,toEl);if(beforeUpdateResult===false){return}else if(beforeUpdateResult instanceof HTMLElement){fromEl=beforeUpdateResult;indexTree(fromEl)}morphAttrs(fromEl,toEl);onElUpdated(fromEl);if(onBeforeElChildrenUpdated(fromEl,toEl)===false){return}}if(fromEl.nodeName!=="TEXTAREA"){morphChildren(fromEl,toEl)}else{specialElHandlers.TEXTAREA(fromEl,toEl)}}function morphChildren(fromEl,toEl){var skipFrom=skipFromChildren(fromEl,toEl);var curToNodeChild=toEl.firstChild;var curFromNodeChild=fromEl.firstChild;var curToNodeKey;var curFromNodeKey;var fromNextSibling;var toNextSibling;var matchingFromEl;outer:while(curToNodeChild){toNextSibling=curToNodeChild.nextSibling;curToNodeKey=getNodeKey(curToNodeChild);while(!skipFrom&&curFromNodeChild){fromNextSibling=curFromNodeChild.nextSibling;if(curToNodeChild.isSameNode&&curToNodeChild.isSameNode(curFromNodeChild)){curToNodeChild=toNextSibling;curFromNodeChild=fromNextSibling;continue outer}curFromNodeKey=getNodeKey(curFromNodeChild);var curFromNodeType=curFromNodeChild.nodeType;var isCompatible=undefined;if(curFromNodeType===curToNodeChild.nodeType){if(curFromNodeType===ELEMENT_NODE){if(curToNodeKey){if(curToNodeKey!==curFromNodeKey){if(matchingFromEl=fromNodesLookup[curToNodeKey]){if(fromNextSibling===matchingFromEl){isCompatible=false}else{fromEl.insertBefore(matchingFromEl,curFromNodeChild);if(curFromNodeKey){addKeyedRemoval(curFromNodeKey)}else{removeNode(curFromNodeChild,fromEl,true)}curFromNodeChild=matchingFromEl;curFromNodeKey=getNodeKey(curFromNodeChild)}}else{isCompatible=false}}}else if(curFromNodeKey){isCompatible=false}isCompatible=isCompatible!==false&&compareNodeNames(curFromNodeChild,curToNodeChild);if(isCompatible){morphEl(curFromNodeChild,curToNodeChild)}}else if(curFromNodeType===TEXT_NODE||curFromNodeType==COMMENT_NODE){isCompatible=true;if(curFromNodeChild.nodeValue!==curToNodeChild.nodeValue){curFromNodeChild.nodeValue=curToNodeChild.nodeValue}}}if(isCompatible){curToNodeChild=toNextSibling;curFromNodeChild=fromNextSibling;continue outer}if(curFromNodeKey){addKeyedRemoval(curFromNodeKey)}else{removeNode(curFromNodeChild,fromEl,true)}curFromNodeChild=fromNextSibling}if(curToNodeKey&&(matchingFromEl=fromNodesLookup[curToNodeKey])&&compareNodeNames(matchingFromEl,curToNodeChild)){if(!skipFrom){addChild(fromEl,matchingFromEl)}morphEl(matchingFromEl,curToNodeChild)}else{var onBeforeNodeAddedResult=onBeforeNodeAdded(curToNodeChild);if(onBeforeNodeAddedResult!==false){if(onBeforeNodeAddedResult){curToNodeChild=onBeforeNodeAddedResult}if(curToNodeChild.actualize){curToNodeChild=curToNodeChild.actualize(fromEl.ownerDocument||doc)}addChild(fromEl,curToNodeChild);handleNodeAdded(curToNodeChild)}}curToNodeChild=toNextSibling;curFromNodeChild=fromNextSibling}cleanupFromEl(fromEl,curFromNodeChild,curFromNodeKey);var specialElHandler=specialElHandlers[fromEl.nodeName];if(specialElHandler){specialElHandler(fromEl,toEl)}}var morphedNode=fromNode;var morphedNodeType=morphedNode.nodeType;var toNodeType=toNode.nodeType;if(!childrenOnly){if(morphedNodeType===ELEMENT_NODE){if(toNodeType===ELEMENT_NODE){if(!compareNodeNames(fromNode,toNode)){onNodeDiscarded(fromNode);morphedNode=moveChildren(fromNode,createElementNS(toNode.nodeName,toNode.namespaceURI))}}else{morphedNode=toNode}}else if(morphedNodeType===TEXT_NODE||morphedNodeType===COMMENT_NODE){if(toNodeType===morphedNodeType){if(morphedNode.nodeValue!==toNode.nodeValue){morphedNode.nodeValue=toNode.nodeValue}return morphedNode}else{morphedNode=toNode}}}if(morphedNode===toNode){onNodeDiscarded(fromNode)}else{if(toNode.isSameNode&&toNode.isSameNode(morphedNode)){return}morphEl(morphedNode,toNode,childrenOnly);if(keyedRemovalList){for(var i=0,len=keyedRemovalList.length;i<len;i++){var elToRemove=fromNodesLookup[keyedRemovalList[i]];if(elToRemove){removeNode(elToRemove,elToRemove.parentNode,false)}}}}if(!childrenOnly&&morphedNode!==fromNode&&fromNode.parentNode){if(morphedNode.actualize){morphedNode=morphedNode.actualize(fromNode.ownerDocument||doc)}fromNode.parentNode.replaceChild(morphedNode,fromNode)}return morphedNode}}var morphdom=morphdomFactory(morphAttrs);return morphdom});

        const TLD = '{{.TLD}}';
        const PORT = {{.Port}};
        const portSuffix = PORT === 80 ? '' : ':' + PORT;
        const INITIAL_DATA = {{.InitialData}};
        let currentApps = [];
        let expandedLogs = null;
        let eventSource = null;
        let claudeEnabled = false;

        // Check if Claude Code integration is configured
        fetch('/api/claude-enabled').then(r => r.json()).then(data => {
            claudeEnabled = data.enabled;
        }).catch(() => { claudeEnabled = false; });

        // Convert name to URL-safe slug (spaces to dashes, lowercase)
        function slugify(name) {
            return name.toLowerCase().replace(/ /g, '-');
        }

        // Theme management
        function getTheme() {
            return document.documentElement.getAttribute('data-theme') || 'system';
        }

        function applyTheme(theme) {
            if (theme === 'system') {
                document.documentElement.removeAttribute('data-theme');
            } else {
                document.documentElement.setAttribute('data-theme', theme);
            }
            updateThemeIcon();
        }

        function setTheme(theme) {
            // Save to server (which broadcasts to other pages)
            fetch('/api/theme', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({theme: theme})
            });
            applyTheme(theme);
        }

        function toggleTheme() {
            const current = getTheme();
            const themes = ['system', 'light', 'dark'];
            const next = themes[(themes.indexOf(current) + 1) % themes.length];
            setTheme(next);
        }

        function updateThemeIcon() {
            const theme = getTheme();
            const icon = document.getElementById('theme-icon');
            if (theme === 'light') icon.textContent = '☀';
            else if (theme === 'dark') icon.textContent = '☾';
            else icon.textContent = '◐';
        }

        // Initialize theme
        setTheme(getTheme());

        // SSE Connection
        function connectSSE() {
            if (eventSource) {
                eventSource.close();
            }

            eventSource = new EventSource('/api/events');

            eventSource.onopen = () => {
                document.getElementById('connection-dot').classList.add('connected');
                document.getElementById('connection-text').textContent = 'Live';
            };

            eventSource.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.type === 'theme') {
                        applyTheme(data.theme);
                    } else if (Array.isArray(data)) {
                        updateApps(data);
                    }
                } catch (e) {
                    console.error('Failed to parse SSE data:', e);
                }
            };

            eventSource.onerror = () => {
                document.getElementById('connection-dot').classList.remove('connected');
                document.getElementById('connection-text').textContent = 'Reconnecting...';
                // EventSource will auto-reconnect
            };
        }

        // Update apps using morphdom for efficient DOM diffing
        function updateApps(newApps) {
            const container = document.getElementById('apps');
            const oldAppsSet = new Set(currentApps.map(a => a.name));
            const newAppsSet = new Set(newApps.map(a => a.name));

            // Generate new HTML
            let newHTML;
            if (!newApps.length) {
                newHTML = ` + "`" + `
                    <div class="empty-state">
                        <h2>No apps configured</h2>
                        <p>Create a config file in <code style="display:inline;padding:2px 6px;margin:0">~/.config/roost-dev/</code></p>
                        <code># Example: create a config for your app
echo "npm run dev" > ~/.config/roost-dev/myapp

# Your command receives a $PORT env var
# Then visit http://myapp.` + "`" + ` + TLD + ` + "`" + `

# For more options, see: roost-dev serve --help</code>
                        <p style="margin-top:20px;font-size:13px">Config directory: <code style="display:inline;padding:2px 6px;margin:0">~/.config/roost-dev/</code></p>
                    </div>
                ` + "`" + `;
            } else {
                newHTML = newApps.map(app => renderApp(app)).join('');
            }

            // Track which apps are newly added for highlight animation (skip on initial load)
            const isInitialLoad = currentApps.length === 0;
            const addedApps = isInitialLoad ? [] : newApps.filter(a => !oldAppsSet.has(a.name)).map(a => a.name);

            // Use morphdom to efficiently update the DOM
            const wrapper = document.createElement('main');
            wrapper.id = 'apps';
            wrapper.innerHTML = newHTML;

            morphdom(container, wrapper, {
                // Preserve scroll position and state
                onBeforeElUpdated(fromEl, toEl) {
                    // Skip updating visible logs panels entirely (preserves button state, scroll, content)
                    if (fromEl.classList?.contains('logs-panel') && fromEl.classList.contains('visible')) {
                        return false;
                    }
                    // Preserve menu visibility
                    if (fromEl.classList?.contains('status-menu') && fromEl.classList.contains('visible')) {
                        toEl.classList.add('visible');
                    }
                    return true;
                },
                onNodeAdded(node) {
                    // Highlight newly added apps
                    if (node.classList?.contains('app')) {
                        const name = node.getAttribute('data-name');
                        if (addedApps.includes(name)) {
                            node.classList.add('highlight');
                            setTimeout(() => node.classList.remove('highlight'), 1000);
                        }
                    }
                    return node;
                }
            });

            currentApps = newApps;
        }

        function renderApp(app) {
            const isRunning = app.running || (app.services && app.services.some(s => s.running));
            const isStarting = app.starting || (app.services && app.services.some(s => s.starting));
            const hasFailed = app.failed || (app.services && app.services.some(s => s.failed));
            const statusClass = hasFailed ? 'failed' : (isRunning ? 'running' : (isStarting ? 'starting' : 'idle'));
            const displayName = app.description || app.name;

            const getServiceStatus = (svc) => svc.failed ? 'failed' : (svc.running ? 'running' : (svc.starting ? 'starting' : 'idle'));

            let servicesHTML = '';
            if (app.services && app.services.length > 0) {
                servicesHTML = ` + "`" + `
                    <div class="services">
                        ${app.services.map(svc => {
                            const svcStatus = getServiceStatus(svc);
                            const svcSlug = slugify(svc.name);
                            const svcName = svcSlug + '-' + app.name;
                            return ` + "`" + `
                            <div class="service">
                                <div class="service-info">
                                    <div class="status-dot-wrapper">
                                        <div class="status-dot ${svcStatus}" onclick="event.stopPropagation(); handleDotClick('${svcName}', event)"></div>
                                        <div class="status-menu" id="menu-${svcName}-active">
                                            <button onclick="event.stopPropagation(); doRestart('${svcName}', event)">Restart</button>
                                            <button class="danger" onclick="event.stopPropagation(); doStop('${svcName}')">Stop</button>
                                        </div>
                                        <div class="status-menu" id="menu-${svcName}-failed">
                                            <button onclick="event.stopPropagation(); doRestart('${svcName}', event)">Restart</button>
                                            <button onclick="event.stopPropagation(); doClear('${svcName}')">Clear</button>
                                        </div>
                                    </div>
                                    <span class="service-name">${svc.name}</span>
                                    ${svc.error ? ` + "`" + `<span class="app-error">${svc.error}</span>` + "`" + ` : ''}
                                </div>
                                <div class="service-meta">
                                    <span class="app-port">${svc.port ? ':' + svc.port : ''}</span>
                                    <span class="app-uptime">${svc.uptime || ''}</span>
                                    <a class="app-url" href="${svc.url}" target="_blank" rel="noopener">
                                        ${svc.url.replace(/^https?:\/\//, '')}
                                    </a>
                                </div>
                            </div>
                        ` + "`" + `}).join('')}
                    </div>
                ` + "`" + `;
            }

            return ` + "`" + `
                <div class="app" data-name="${app.name}">
                    <div class="app-header" onclick="toggleLogs('${app.name}')">
                        <div class="app-info">
                            <div class="status-dot-wrapper">
                                <div class="status-dot ${statusClass}" onclick="event.stopPropagation(); handleDotClick('${app.name}', event)"></div>
                                <div class="status-menu" id="menu-${app.name}-active">
                                    <button onclick="event.stopPropagation(); doRestart('${app.name}', event)">Restart</button>
                                    <button class="danger" onclick="event.stopPropagation(); doStop('${app.name}')">Stop</button>
                                </div>
                                <div class="status-menu" id="menu-${app.name}-failed">
                                    <button onclick="event.stopPropagation(); doRestart('${app.name}', event)">Restart</button>
                                    <button onclick="event.stopPropagation(); doClear('${app.name}')">Clear</button>
                                </div>
                            </div>
                            <span class="app-name">${displayName}</span>
                            ${app.aliases && app.aliases.length ? ` + "`" + `<span class="app-aliases">aka ${app.aliases.join(', ')}</span>` + "`" + ` : ''}
                        </div>
                        <div class="app-meta">
                            <span class="app-port">${app.port ? ':' + app.port : ''}</span>
                            <span class="app-uptime">${app.uptime || ''}</span>
                            ${(!(app.services && app.services.length) || (app.services && app.services.some(s => s.default))) ? ` + "`" + `<a class="app-url" href="${app.url}" target="_blank" rel="noopener" onclick="event.stopPropagation()">
                                ${app.name}.` + "`" + ` + TLD + ` + "`" + `
                            </a>` + "`" + ` : ''}
                        </div>
                    </div>
                    ${servicesHTML}
                    <div class="logs-panel" id="logs-${app.name}">
                        <div class="logs-header">
                            <span class="logs-title">Logs</span>
                            <div class="logs-actions" id="logs-actions-${app.name}">
                                <button class="copy-btn" onclick="event.stopPropagation(); copyLogs('${app.name}', event)" title="Copy logs">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>
                                </button>
                                ${hasFailed ? ` + "`" + `<button class="agent-btn" onclick="event.stopPropagation(); copyForAgent('${app.name}', event)" title="Copy for agent">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect><path d="M9 14h.01M15 14h.01M10 18c.5.3 1.2.5 2 .5s1.5-.2 2-.5"></path></svg>
                                </button>` + "`" + ` : ''}
                                ${hasFailed && claudeEnabled ? ` + "`" + `<button class="claude-btn" onclick="event.stopPropagation(); fixWithClaudeCode('${app.name}', event)" title="Fix with Claude Code">
                                    <svg viewBox="0 0 16 16" fill="currentColor"><path d="m3.127 10.604 3.135-1.76.053-.153-.053-.085H6.11l-.525-.032-1.791-.048-1.554-.065-1.505-.08-.38-.081L0 7.832l.036-.234.32-.214.455.04 1.009.069 1.513.105 1.097.064 1.626.17h.259l.036-.105-.089-.065-.068-.064-1.566-1.062-1.695-1.121-.887-.646-.48-.327-.243-.306-.104-.67.435-.48.585.04.15.04.593.456 1.267.981 1.654 1.218.242.202.097-.068.012-.049-.109-.181-.9-1.626-.96-1.655-.428-.686-.113-.411a2 2 0 0 1-.068-.484l.496-.674L4.446 0l.662.089.279.242.411.94.666 1.48 1.033 2.014.302.597.162.553.06.17h.105v-.097l.085-1.134.157-1.392.154-1.792.052-.504.25-.605.497-.327.387.186.319.456-.045.294-.19 1.23-.37 1.93-.243 1.29h.142l.161-.16.654-.868 1.097-1.372.484-.545.565-.601.363-.287h.686l.505.751-.226.775-.707.895-.585.759-.839 1.13-.524.904.048.072.125-.012 1.897-.403 1.024-.186 1.223-.21.553.258.06.263-.218.536-1.307.323-1.533.307-2.284.54-.028.02.032.04 1.029.098.44.024h1.077l2.005.15.525.346.315.424-.053.323-.807.411-3.631-.863-.872-.218h-.12v.073l.726.71 1.331 1.202 1.667 1.55.084.383-.214.302-.226-.032-1.464-1.101-.565-.497-1.28-1.077h-.084v.113l.295.432 1.557 2.34.08.718-.112.234-.404.141-.444-.08-.911-1.28-.94-1.44-.759-1.291-.093.053-.448 4.821-.21.246-.484.186-.403-.307-.214-.496.214-.98.258-1.28.21-1.016.19-1.263.112-.42-.008-.028-.092.012-.953 1.307-1.448 1.957-1.146 1.227-.274.109-.477-.247.045-.44.266-.39 1.586-2.018.956-1.25.617-.723-.004-.105h-.036l-4.212 2.736-.75.096-.324-.302.04-.496.154-.162 1.267-.871z"/></svg>
                                </button>` + "`" + ` : ''}
                                <button class="clear-btn" onclick="event.stopPropagation(); clearLogs('${app.name}')" title="Clear logs">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                                </button>
                            </div>
                        </div>
                        <div class="logs-content" id="logs-content-${app.name}"></div>
                    </div>
                </div>
            ` + "`" + `;
        }

        async function toggleLogs(name) {
            const panel = document.getElementById('logs-' + name);
            const isVisible = panel.classList.contains('visible');

            // Hide all panels
            document.querySelectorAll('.logs-panel').forEach(p => p.classList.remove('visible'));

            if (!isVisible) {
                panel.classList.add('visible');
                expandedLogs = name;
                // Check if this app has failed to trigger AI analysis
                const app = currentApps.find(a => a.name === name);
                const hasFailed = app && (app.failed || (app.services && app.services.some(s => s.failed)));
                await fetchLogs(name, hasFailed);
            } else {
                expandedLogs = null;
            }
        }

        // Check if user has selected text within an element
        function hasSelectionIn(el) {
            const sel = window.getSelection();
            if (!sel || sel.isCollapsed || !sel.rangeCount) return false;
            const range = sel.getRangeAt(0);
            return el.contains(range.commonAncestorContainer);
        }

        // Convert ANSI escape codes to HTML spans
        function ansiToHtml(text) {
            const colors = {
                '30': '#000', '31': '#e74c3c', '32': '#2ecc71', '33': '#f1c40f',
                '34': '#3498db', '35': '#9b59b6', '36': '#1abc9c', '37': '#ecf0f1',
                '90': '#7f8c8d', '91': '#e74c3c', '92': '#2ecc71', '93': '#f1c40f',
                '94': '#3498db', '95': '#9b59b6', '96': '#1abc9c', '97': '#fff'
            };
            let result = '';
            let i = 0;
            let openSpans = 0;
            while (i < text.length) {
                if (text[i] === '\x1b' && text[i+1] === '[') {
                    let j = i + 2;
                    while (j < text.length && text[j] !== 'm') j++;
                    const codes = text.slice(i+2, j).split(';');
                    i = j + 1;
                    for (const code of codes) {
                        if (code === '0' || code === '39' || code === '22' || code === '23') {
                            if (openSpans > 0) { result += '</span>'; openSpans--; }
                        } else if (colors[code]) {
                            result += '<span style="color:' + colors[code] + '">';
                            openSpans++;
                        } else if (code === '1') {
                            result += '<span style="font-weight:bold">';
                            openSpans++;
                        } else if (code === '3') {
                            result += '<span style="font-style:italic">';
                            openSpans++;
                        }
                    }
                } else {
                    // Escape HTML
                    const c = text[i];
                    if (c === '<') result += '&lt;';
                    else if (c === '>') result += '&gt;';
                    else if (c === '&') result += '&amp;';
                    else result += c;
                    i++;
                }
            }
            while (openSpans-- > 0) result += '</span>';
            return result;
        }

        // Analyze logs with Ollama to find error lines (async, updates UI when done)
        async function analyzeLogsWithAI(name, lines) {
            try {
                const res = await fetch('/api/analyze-logs?name=' + encodeURIComponent(name));
                const data = await res.json();
                if (!data.enabled || data.error || !data.errorLines || data.errorLines.length === 0) {
                    return;
                }
                const content = document.getElementById('logs-content-' + name);
                if (!content || hasSelectionIn(content)) return;
                const errorSet = new Set(data.errorLines);
                const highlighted = lines.map((line, idx) => {
                    const html = ansiToHtml(line);
                    return errorSet.has(idx) ? '<mark>' + html + '</mark>' : html;
                }).join('\n');
                content.innerHTML = highlighted;
            } catch (e) {
                console.log('AI analysis skipped:', e);
            }
        }

        async function fetchLogs(name, triggerAnalysis = false) {
            try {
                const res = await fetch('/api/logs?name=' + encodeURIComponent(name));
                const lines = await res.json();
                const content = document.getElementById('logs-content-' + name);
                if (content) {
                    // Skip update if user is selecting text in logs
                    if (hasSelectionIn(content)) return;
                    const wasAtBottom = content.scrollHeight - content.scrollTop <= content.clientHeight + 50;
                    content.innerHTML = ansiToHtml((lines || []).join('\n'));
                    if (wasAtBottom) {
                        content.scrollTop = content.scrollHeight;
                    }
                    // Show/hide buttons based on log content
                    updateLogButtons(name, lines && lines.length > 0);
                    // Trigger AI analysis for failed apps
                    if (triggerAnalysis && lines && lines.length > 0) {
                        analyzeLogsWithAI(name, lines);
                    }
                }
            } catch (e) {
                console.error('Failed to fetch logs:', e);
            }
        }

        function updateLogButtons(name, hasLogs) {
            const actions = document.getElementById('logs-actions-' + name);
            if (!actions) return;
            const copyBtn = actions.querySelector('.copy-btn');
            const clearBtn = actions.querySelector('.clear-btn');
            const agentBtn = actions.querySelector('.agent-btn');
            const claudeBtn = actions.querySelector('.claude-btn');
            if (copyBtn) copyBtn.classList.toggle('visible', hasLogs);
            if (clearBtn) clearBtn.classList.toggle('visible', hasLogs);
            if (agentBtn) agentBtn.classList.toggle('visible', hasLogs);
            if (claudeBtn) claudeBtn.classList.toggle('visible', hasLogs);
        }

        function clearLogs(name) {
            const content = document.getElementById('logs-content-' + name);
            if (content) content.textContent = '';
            updateLogButtons(name, false);
        }

        function getLogsText(name) {
            const content = document.getElementById('logs-content-' + name);
            // Use selection if user has selected text within logs
            if (hasSelectionIn(content)) {
                return window.getSelection().toString();
            }
            return content.textContent;
        }

        function copyLogs(name, event) {
            const text = getLogsText(name);

            // We use execCommand('copy') instead of navigator.clipboard.writeText() because
            // the Clipboard API requires a "secure context" (HTTPS or literal localhost).
            // Even though *.test domains resolve to 127.0.0.1, browsers check the hostname,
            // not the resolved IP - so roost-dev.test is considered insecure.
            // execCommand('copy') is deprecated but works in non-secure contexts by copying
            // selected text from a DOM element (hence the hidden textarea trick).
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);

            const btn = event.target.closest('button');
            const origHTML = btn.innerHTML;
            btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
            setTimeout(() => btn.innerHTML = origHTML, 500);
        }

        function copyForAgent(name, event) {
            const logs = getLogsText(name);
            const app = currentApps.find(a => a.name === name);
            const hasFailed = app && (app.failed || (app.services && app.services.some(s => s.failed)));

            const bt = String.fromCharCode(96);
            let context = 'I am using roost-dev, a local development server that manages apps via config files in ~/.config/roost-dev/.\n\n';
            if (hasFailed) {
                context += 'The app "' + name + '" failed to start. ';
            } else {
                context += 'The app "' + name + '" is having issues. ';
            }
            context += 'The config file is at:\n~/.config/roost-dev/' + name + '.yml\n\n' +
                'Here are the logs:\n\n' +
                bt+bt+bt + '\n' + logs + '\n' + bt+bt+bt + '\n\n' +
                'To restart the app: roost-dev restart ' + name + '\n' +
                'To learn more about roost-dev commands: roost-dev --help\n\n' +
                'Please help me understand and fix this error.';

            const textarea = document.createElement('textarea');
            textarea.value = context;
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);

            const btn = event.target.closest('button');
            const origHTML = btn.innerHTML;
            btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
            setTimeout(() => btn.innerHTML = origHTML, 500);
        }

        async function fixWithClaudeCode(name, event) {
            const btn = event.target.closest('button');
            const origHTML = btn.innerHTML;
            btn.disabled = true;
            try {
                const res = await fetch('/api/open-terminal?name=' + encodeURIComponent(name));
                if (!res.ok) {
                    const err = await res.text();
                    console.error('Failed to open terminal:', err);
                    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>';
                    setTimeout(() => {
                        btn.innerHTML = origHTML;
                        btn.disabled = false;
                    }, 2000);
                    return;
                }
                btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
                setTimeout(() => {
                    btn.innerHTML = origHTML;
                    btn.disabled = false;
                }, 1000);
            } catch (e) {
                console.error('Failed to open terminal:', e);
                btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>';
                setTimeout(() => {
                    btn.innerHTML = origHTML;
                    btn.disabled = false;
                }, 2000);
            }
        }

        async function stop(name) {
            await fetch('/api/stop?name=' + encodeURIComponent(name));
        }

        async function restart(name) {
            await fetch('/api/restart?name=' + encodeURIComponent(name));
        }

        async function start(name) {
            await fetch('/api/start?name=' + encodeURIComponent(name));
        }

        function closeAllMenus() {
            document.querySelectorAll('.status-menu').forEach(m => m.classList.remove('visible'));
        }

        function handleDotClick(name, event) {
            event.stopPropagation();
            closeAllMenus();

            const dot = event.target;
            const isRunning = dot.classList.contains('running');
            const isStarting = dot.classList.contains('starting');
            const isFailed = dot.classList.contains('failed');

            if (isRunning || isStarting) {
                const menu = document.getElementById('menu-' + name + '-active');
                if (menu) menu.classList.add('visible');
            } else if (isFailed) {
                const menu = document.getElementById('menu-' + name + '-failed');
                if (menu) menu.classList.add('visible');
            } else {
                dot.className = 'status-dot starting';
                start(name);
            }
        }

        async function doRestart(name, event) {
            closeAllMenus();
            const wrapper = event.target.closest('.status-dot-wrapper');
            const dot = wrapper ? wrapper.querySelector('.status-dot') : null;
            if (dot) dot.className = 'status-dot starting';
            await restart(name);
        }

        async function doStop(name) {
            closeAllMenus();
            await stop(name);
        }

        async function doClear(name) {
            closeAllMenus();
            // Stop clears the failed state and turns it grey
            await stop(name);
        }

        document.addEventListener('click', closeAllMenus);

        // Periodically refresh logs if panel is open
        setInterval(() => {
            if (expandedLogs) {
                fetchLogs(expandedLogs);
            }
        }, 2000);

        // Render initial data immediately, then connect SSE for updates
        updateApps(INITIAL_DATA || []);
        connectSSE();
    </script>
</body>
</html>
`
